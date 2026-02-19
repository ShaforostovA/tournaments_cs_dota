package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func openTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping db: %v", err)
	}

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		_ = db.Close()
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		_ = db.Close()
		t.Fatalf("set search_path: %v", err)
	}

	applyMigrations(t, db)

	cleanup := func() {
		_, _ = db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = db.Close()
	}

	return db, cleanup
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve current file path")
	}
	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		files = append(files, filepath.Join(migrationsDir, entry.Name()))
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}

	for _, path := range files {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if strings.TrimSpace(string(sqlBytes)) == "" {
			continue
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
}

func createTournament(t *testing.T, db *sql.DB, allowOdd bool) int64 {
	t.Helper()

	const query = `
INSERT INTO tournaments (game, name, allow_odd, status, is_listed, is_bracket_published)
VALUES ('DOTA2', 'Test', $1, 'DRAFT', TRUE, TRUE)
RETURNING id`
	var id int64
	if err := db.QueryRow(query, allowOdd).Scan(&id); err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	return id
}

func createTeam(t *testing.T, db *sql.DB, tournamentID int64, name string, seed int) int64 {
	t.Helper()
	const query = `
INSERT INTO teams (tournament_id, name, status, seed)
VALUES ($1, $2, 'ACTIVE', $3)
RETURNING id`
	var id int64
	if err := db.QueryRow(query, tournamentID, name, seed).Scan(&id); err != nil {
		t.Fatalf("create team: %v", err)
	}
	return id
}

func TestGenerateBracketWithByesAutoAdvance(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	tournamentID := createTournament(t, db, true)
	for i := 1; i <= 5; i++ {
		createTeam(t, db, tournamentID, fmt.Sprintf("Team %d", i), i)
	}

	resp, err := generateSingleEliminationBracket(context.Background(), db, strconv.FormatInt(tournamentID, 10), false)
	if err != nil {
		t.Fatalf("generate bracket: %v", err)
	}
	if resp.BracketSize != 8 {
		t.Fatalf("expected bracket size 8, got %d", resp.BracketSize)
	}
	if resp.ByeSlots != 3 {
		t.Fatalf("expected 3 byes, got %d", resp.ByeSlots)
	}

	const byesQuery = `
SELECT COUNT(*)
FROM matches
WHERE tournament_id = $1
  AND round = 1
  AND status = 'FINISHED'
  AND ((team_a_id IS NULL AND team_b_id IS NOT NULL) OR (team_b_id IS NULL AND team_a_id IS NOT NULL))`
	var byeFinished int
	if err := db.QueryRow(byesQuery, tournamentID).Scan(&byeFinished); err != nil {
		t.Fatalf("count bye matches: %v", err)
	}
	if byeFinished != 3 {
		t.Fatalf("expected 3 finished bye matches, got %d", byeFinished)
	}

	const slotsQuery = `
SELECT COALESCE(SUM((team_a_id IS NOT NULL)::int + (team_b_id IS NOT NULL)::int), 0)
FROM matches
WHERE tournament_id = $1 AND round = 2`
	var filledSlots int
	if err := db.QueryRow(slotsQuery, tournamentID).Scan(&filledSlots); err != nil {
		t.Fatalf("count round2 slots: %v", err)
	}
	if filledSlots != 3 {
		t.Fatalf("expected 3 filled slots in round2, got %d", filledSlots)
	}
}

func TestAdvanceWinnerThroughBracket(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	tournamentID := createTournament(t, db, false)
	for i := 1; i <= 4; i++ {
		createTeam(t, db, tournamentID, fmt.Sprintf("Team %d", i), i)
	}

	if _, err := generateSingleEliminationBracket(context.Background(), db, strconv.FormatInt(tournamentID, 10), false); err != nil {
		t.Fatalf("generate bracket: %v", err)
	}

	if _, err := db.Exec(`UPDATE tournaments SET status = 'RUNNING' WHERE id = $1`, tournamentID); err != nil {
		t.Fatalf("set tournament running: %v", err)
	}

	const matchQuery = `
SELECT id, team_a_id, team_b_id
FROM matches
WHERE tournament_id = $1 AND round = 1 AND index_in_round = 1`
	var matchID int64
	var teamAID sql.NullInt64
	var teamBID sql.NullInt64
	if err := db.QueryRow(matchQuery, tournamentID).Scan(&matchID, &teamAID, &teamBID); err != nil {
		t.Fatalf("select match: %v", err)
	}
	if !teamAID.Valid || !teamBID.Valid {
		t.Fatalf("expected both teams assigned in round1")
	}

	payload := updateMatchResultRequest{
		ScoreA:       1,
		ScoreB:       0,
		WinnerTeamID: teamAID.Int64,
	}
	if _, err := updateMatchResult(context.Background(), db, strconv.FormatInt(tournamentID, 10), strconv.FormatInt(matchID, 10), payload); err != nil {
		t.Fatalf("update match result: %v", err)
	}

	const nextQuery = `
SELECT team_a_id
FROM matches
WHERE tournament_id = $1 AND round = 2 AND index_in_round = 1`
	var nextTeamA sql.NullInt64
	if err := db.QueryRow(nextQuery, tournamentID).Scan(&nextTeamA); err != nil {
		t.Fatalf("select next match: %v", err)
	}
	if !nextTeamA.Valid || nextTeamA.Int64 != teamAID.Int64 {
		t.Fatalf("expected winner advanced to next round slot")
	}
}
