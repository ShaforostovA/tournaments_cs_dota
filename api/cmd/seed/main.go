package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"tournaments/api/internal/db"
)

type tournamentSeed struct {
	Game        string
	Name        string
	Description string
	StartDate   *string
	EndDate     *string
	Status      string
	IsListed    bool
}

func main() {
	conn, err := db.Open()
	if err != nil {
		log.Fatalf("db connection error: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := seed(ctx, conn); err != nil {
		log.Fatalf("seed error: %v", err)
	}

	log.Println("seed completed")
}

func seed(ctx context.Context, conn *sql.DB) error {
	seeds := buildSeeds()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := clearData(ctx, tx); err != nil {
		return err
	}

	if err := insertAdmin(ctx, tx); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	for key, t := range seeds {
		_, err := insertTournament(ctx, tx, t)
		if err != nil {
			return fmt.Errorf("insert tournament %s: %w", key, err)
		}
	}

	return tx.Commit()
}

func clearData(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		"DELETE FROM audit_log",
		"DELETE FROM schedule",
		"DELETE FROM matches",
		"DELETE FROM teams",
		"DELETE FROM tournaments",
		"DELETE FROM users",
	}
	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func insertAdmin(ctx context.Context, tx *sql.Tx) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO users (login, password_hash)
VALUES ($1,$2)`
	_, err = tx.ExecContext(ctx, q, "admin", string(passwordHash))
	return err
}

func insertTournament(ctx context.Context, tx *sql.Tx, t tournamentSeed) (int64, error) {
	const q = `
INSERT INTO tournaments (
  game, name, description, start_date, end_date, allow_odd, status,
  is_listed, is_bracket_published, schedule_visibility_ahead
) VALUES ($1,$2,$3,$4,$5,TRUE,$6,$7,FALSE,'0')
RETURNING id`

	var id int64
	var startDate any
	var endDate any
	if t.StartDate != nil {
		startDate = *t.StartDate
	}
	if t.EndDate != nil {
		endDate = *t.EndDate
	}
	row := tx.QueryRowContext(ctx, q, t.Game, t.Name, t.Description, startDate, endDate, t.Status, t.IsListed)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func buildSeeds() map[string]tournamentSeed {
	return map[string]tournamentSeed{}
}
