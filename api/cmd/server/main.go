package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"tournaments/api/internal/db"
)

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type tournament struct {
	ID                      int64   `json:"id"`
	Game                    string  `json:"game"`
	Name                    string  `json:"name"`
	Description             *string `json:"description,omitempty"`
	StartDate               *string `json:"startDate,omitempty"`
	EndDate                 *string `json:"endDate,omitempty"`
	AllowOdd                bool    `json:"allowOdd"`
	Status                  string  `json:"status"`
	IsListed                bool    `json:"isListed"`
	IsBracketPublished      bool    `json:"isBracketPublished"`
	ScheduleVisibilityAhead string  `json:"scheduleVisibilityAhead"`
	CreatedAt               string  `json:"createdAt"`
	UpdatedAt               string  `json:"updatedAt"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type createTournamentRequest struct {
	Game string `json:"game"`
	Name string `json:"name"`
}

type updateTournamentRequest struct {
	Game        *string `json:"game"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
	AllowOdd    *bool   `json:"allowOdd"`
}

type team struct {
	ID           int64   `json:"id"`
	TournamentID int64   `json:"tournamentId"`
	Name         string  `json:"name"`
	Note         *string `json:"note,omitempty"`
	Status       string  `json:"status"`
	StatusReason *string `json:"statusReason,omitempty"`
	Seed         *int    `json:"seed,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type createTeamRequest struct {
	Name string  `json:"name"`
	Note *string `json:"note"`
}

type updateTeamRequest struct {
	Name *string `json:"name"`
	Note *string `json:"note"`
}

type generateSeedingRequest struct {
	Overwrite bool `json:"overwrite"`
}

type reorderSeedingRequest struct {
	TeamIDs []int64 `json:"teamIds"`
}

type generateBracketRequest struct {
	Overwrite bool `json:"overwrite"`
}

type generateScheduleRequest struct {
	Overwrite bool `json:"overwrite"`
}

type reorderScheduleRequest struct {
	MatchIDs []int64 `json:"matchIds"`
}

type updateMatchSidesRequest struct {
	Mode      string `json:"mode"`
	TeamASide string `json:"teamASide"`
	TeamBSide string `json:"teamBSide"`
}

type updateMatchStatusRequest struct {
	Action string `json:"action"`
}

type updateMatchResultRequest struct {
	ScoreA       int   `json:"scoreA"`
	ScoreB       int   `json:"scoreB"`
	WinnerTeamID int64 `json:"winnerTeamId"`
}

type updateMatchScoreRequest struct {
	ScoreA int `json:"scoreA"`
	ScoreB int `json:"scoreB"`
}

type updateTeamStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type forfeitMatchRequest struct {
	WinnerTeamID int64  `json:"winnerTeamId"`
	Reason       string `json:"reason"`
}

type adminInfo struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type adminClaims struct {
	AdminID int64  `json:"adminId"`
	Login   string `json:"login"`
	jwt.RegisteredClaims
}

type rateBucket struct {
	Tokens     int
	LastRefill time.Time
}

type adminTournamentListResponse struct {
	Items      []tournament `json:"items"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Total      int          `json:"total"`
	TotalPages int          `json:"totalPages"`
}

type adminTeamListResponse struct {
	Items      []team `json:"items"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	Total      int    `json:"total"`
	TotalPages int    `json:"totalPages"`
}

type importCSVError struct {
	Row     int    `json:"row"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type importCSVResponse struct {
	Mode       string           `json:"mode"`
	Total      int              `json:"total"`
	Created    int              `json:"created"`
	Duplicates int              `json:"duplicates"`
	Errors     []importCSVError `json:"errors"`
}

type auditLogEntry struct {
	ID           int64           `json:"id"`
	UserID       *int64          `json:"userId,omitempty"`
	UserLogin    *string         `json:"userLogin,omitempty"`
	Entity       string          `json:"entity"`
	EntityID     int64           `json:"entityId"`
	Action       string          `json:"action"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    string          `json:"createdAt"`
	TournamentID int64           `json:"tournamentId"`
}

type auditLogListResponse struct {
	Items      []auditLogEntry `json:"items"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	Total      int             `json:"total"`
	TotalPages int             `json:"totalPages"`
}

type teamValidationResponse struct {
	TournamentID    int64   `json:"tournamentId"`
	AllowOdd        bool    `json:"allowOdd"`
	TeamCount       int     `json:"teamCount"`
	IsValid         bool    `json:"isValid"`
	Message         string  `json:"message"`
	SuggestedAction *string `json:"suggestedAction,omitempty"`
	BracketSize     *int    `json:"bracketSize,omitempty"`
}

type generateSeedingResponse struct {
	TournamentID int64 `json:"tournamentId"`
	TeamCount    int   `json:"teamCount"`
	UpdatedCount int   `json:"updatedCount"`
}

type bracketRoundInfo struct {
	Round          int `json:"round"`
	MatchesInRound int `json:"matchesInRound"`
}

type generateBracketResponse struct {
	TournamentID  int64              `json:"tournamentId"`
	BracketSize   int                `json:"bracketSize"`
	RoundsCount   int                `json:"roundsCount"`
	TotalMatches  int                `json:"totalMatches"`
	ByeSlots      int                `json:"byeSlots"`
	TournamentNow string             `json:"tournamentNow"`
	Rounds        []bracketRoundInfo `json:"rounds"`
}

type scheduleItem struct {
	ID           int64   `json:"id"`
	TournamentID int64   `json:"tournamentId"`
	MatchID      int64   `json:"matchId"`
	Position     int     `json:"position"`
	MatchRound   int     `json:"matchRound"`
	MatchIndex   int     `json:"matchIndex"`
	Status       string  `json:"status"`
	Bo           int     `json:"bo"`
	TeamAID      *int64  `json:"teamAId,omitempty"`
	TeamBID      *int64  `json:"teamBId,omitempty"`
	TeamAName    *string `json:"teamAName,omitempty"`
	TeamBName    *string `json:"teamBName,omitempty"`
	ScoreA       int     `json:"scoreA"`
	ScoreB       int     `json:"scoreB"`
	WinnerTeamID *int64  `json:"winnerTeamId,omitempty"`
	SideMode     string  `json:"sideMode"`
	TeamASide    string  `json:"teamASide"`
	TeamBSide    string  `json:"teamBSide"`
}

type scheduleListResponse struct {
	Items      []scheduleItem `json:"items"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	Total      int            `json:"total"`
	TotalPages int            `json:"totalPages"`
}

type publicScheduleItem struct {
	Position     int     `json:"position"`
	MatchID      int64   `json:"matchId"`
	Round        int     `json:"round"`
	IndexInRound int     `json:"indexInRound"`
	Status       string  `json:"status"`
	TeamAName    *string `json:"teamAName,omitempty"`
	TeamBName    *string `json:"teamBName,omitempty"`
	ScoreA       int     `json:"scoreA"`
	ScoreB       int     `json:"scoreB"`
	SideMode     string  `json:"sideMode"`
	TeamASide    string  `json:"teamASide"`
	TeamBSide    string  `json:"teamBSide"`
}

type publicScheduleResponse struct {
	TournamentID int64                `json:"tournamentId"`
	TotalVisible int                  `json:"totalVisible"`
	Items        []publicScheduleItem `json:"items"`
}

type activeMatchItem struct {
	MatchID        int64   `json:"matchId"`
	TournamentID   int64   `json:"tournamentId"`
	TournamentName string  `json:"tournamentName"`
	Game           string  `json:"game"`
	Round          int     `json:"round"`
	IndexInRound   int     `json:"indexInRound"`
	Status         string  `json:"status"`
	TeamAName      *string `json:"teamAName,omitempty"`
	TeamBName      *string `json:"teamBName,omitempty"`
	ScoreA         int     `json:"scoreA"`
	ScoreB         int     `json:"scoreB"`
	StartsAt       *string `json:"startsAt,omitempty"`
}

type activeMatchesResponse struct {
	Items []activeMatchItem `json:"items"`
}

type publicBracketMatch struct {
	ID           int64   `json:"id"`
	Round        int     `json:"round"`
	IndexInRound int     `json:"indexInRound"`
	Status       string  `json:"status"`
	Bo           int     `json:"bo"`
	DisputeRule  string  `json:"disputeRule"`
	TeamAID      *int64  `json:"teamAId,omitempty"`
	TeamAName    *string `json:"teamAName,omitempty"`
	TeamBID      *int64  `json:"teamBId,omitempty"`
	TeamBName    *string `json:"teamBName,omitempty"`
	ScoreA       int     `json:"scoreA"`
	ScoreB       int     `json:"scoreB"`
	WinnerTeamID *int64  `json:"winnerTeamId,omitempty"`
}

type publicBracketRound struct {
	Round   int                  `json:"round"`
	Matches []publicBracketMatch `json:"matches"`
}

type publicBracketResponse struct {
	TournamentID int64                `json:"tournamentId"`
	Rounds       []publicBracketRound `json:"rounds"`
}

type generateScheduleResponse struct {
	TournamentID int64 `json:"tournamentId"`
	TotalMatches int   `json:"totalMatches"`
	UpdatedCount int   `json:"updatedCount"`
}

type roundSetting struct {
	Round       int    `json:"round"`
	Bo          int    `json:"bo"`
	DisputeRule string `json:"disputeRule"`
	Matches     int    `json:"matches"`
}

type updateRoundSettingRequest struct {
	Bo          int    `json:"bo"`
	DisputeRule string `json:"disputeRule"`
}

type updateVisibilityRequest struct {
	IsBracketPublished bool   `json:"isBracketPublished"`
	ScheduleVisibility string `json:"scheduleVisibilityAhead"`
}

type matchStatusResponse struct {
	TournamentID int64   `json:"tournamentId"`
	MatchID      int64   `json:"matchId"`
	Status       string  `json:"status"`
	StartsAt     *string `json:"startsAt,omitempty"`
	UpdatedAt    string  `json:"updatedAt"`
}

type matchResultResponse struct {
	TournamentID int64   `json:"tournamentId"`
	MatchID      int64   `json:"matchId"`
	Status       string  `json:"status"`
	ScoreA       int     `json:"scoreA"`
	ScoreB       int     `json:"scoreB"`
	WinnerTeamID int64   `json:"winnerTeamId"`
	EndedAt      *string `json:"endedAt,omitempty"`
	UpdatedAt    string  `json:"updatedAt"`
}

type matchScoreResponse struct {
	TournamentID int64  `json:"tournamentId"`
	MatchID      int64  `json:"matchId"`
	Status       string `json:"status"`
	ScoreA       int    `json:"scoreA"`
	ScoreB       int    `json:"scoreB"`
	UpdatedAt    string `json:"updatedAt"`
}

type realtimeEvent struct {
	Type         string `json:"type"`
	TournamentID int64  `json:"tournamentId"`
	OccurredAt   string `json:"occurredAt"`
}

type wsHub struct {
	mu      sync.Mutex
	clients map[int64]map[*websocket.Conn]struct{}
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type loginLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		buckets: make(map[string]*rateBucket),
	}
}

func (l *loginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	maxTokens := getenvInt("LOGIN_RATE_LIMIT", 5)
	if maxTokens < 1 {
		maxTokens = 1
	}
	refillInterval := getenvDuration("LOGIN_RATE_WINDOW", time.Minute)
	if refillInterval <= 0 {
		refillInterval = time.Minute
	}

	bucket, ok := l.buckets[key]
	now := time.Now()
	if !ok {
		l.buckets[key] = &rateBucket{Tokens: maxTokens - 1, LastRefill: now}
		return true
	}

	if now.Sub(bucket.LastRefill) >= refillInterval {
		bucket.Tokens = maxTokens
		bucket.LastRefill = now
	}

	if bucket.Tokens <= 0 {
		return false
	}
	bucket.Tokens--
	return true
}

type clientError struct {
	Status  int
	Message string
}

func (e clientError) Error() string {
	return e.Message
}

func main() {
	r := chi.NewRouter()
	r.Use(corsMiddleware)

	conn, err := db.Open()
	if err != nil {
		log.Fatalf("db connection error: %v", err)
	}
	defer conn.Close()
	realtimeHub := newWSHub()
	loginLimiter := newLoginLimiter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ready", Time: time.Now().UTC().Format(time.RFC3339)})
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		tournamentID, err := parseTournamentIDParam(r.URL.Query().Get("tournamentId"))
		if err != nil {
			http.Error(w, "???????????? id ???????", http.StatusBadRequest)
			return
		}
		if err := realtimeHub.serveWS(w, r, tournamentID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	})

	r.Get("/api/v1/tournaments", func(w http.ResponseWriter, r *http.Request) {
		game := r.URL.Query().Get("game")
		items, err := listTournaments(r.Context(), conn, game)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	r.Get("/api/v1/tournaments/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		item, err := getTournament(r.Context(), conn, id)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Get("/api/v1/tournaments/{id}/bracket", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		item, err := getTournament(r.Context(), conn, id)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bracket, err := getPublicBracket(r.Context(), conn, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		bracket.TournamentID = item.ID
		writeJSON(w, http.StatusOK, bracket)
	})

	r.Get("/api/v1/tournaments/{id}/schedule", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		item, err := getTournament(r.Context(), conn, id)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		schedule, err := getPublicSchedule(r.Context(), conn, item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, schedule)
	})

	r.Get("/api/v1/matches/active", func(w http.ResponseWriter, r *http.Request) {
		game := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("game")))
		if game != "" && game != "DOTA2" && game != "CS2" {
			http.Error(w, "???? ?? ??????????????", http.StatusBadRequest)
			return
		}

		var tournamentID int64
		if raw := strings.TrimSpace(r.URL.Query().Get("tournamentId")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || parsed < 1 {
				http.Error(w, "???????????? id ???????", http.StatusBadRequest)
				return
			}
			tournamentID = parsed
		}

		items, err := listActiveMatches(r.Context(), conn, game, tournamentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, activeMatchesResponse{Items: items})
	})

	r.Post("/api/v1/admin/login", func(w http.ResponseWriter, r *http.Request) {
		clientKey := clientIP(r)
		if !loginLimiter.Allow(clientKey) {
			http.Error(w, "??????? ????? ??????? ?????", http.StatusTooManyRequests)
			return
		}
		var payload loginRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}
		payload.Login = strings.TrimSpace(payload.Login)
		if payload.Login == "" || payload.Password == "" {
			http.Error(w, "????? ? ?????? ???????????", http.StatusBadRequest)
			return
		}

		user, err := findUserByLogin(r.Context(), conn, payload.Login)
		if err == sql.ErrNoRows {
			http.Error(w, "???????? ??????? ??????", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
			http.Error(w, "???????? ??????? ??????", http.StatusUnauthorized)
			return
		}

		token, err := buildAdminToken(user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, token)
		writeJSON(w, http.StatusOK, adminInfo{ID: user.ID, Login: user.Login})
	})

	r.Post("/api/v1/admin/logout", func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/api/v1/admin/me", func(w http.ResponseWriter, r *http.Request) {
		info, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, info)
	})

	r.Get("/api/v1/admin/tournaments", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		game := strings.TrimSpace(r.URL.Query().Get("game"))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		search := strings.TrimSpace(r.URL.Query().Get("search"))
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 10)
		if pageSize > 50 {
			pageSize = 50
		}

		result, err := listAdminTournaments(r.Context(), conn, game, status, search, page, pageSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	r.Post("/api/v1/admin/tournaments", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		var payload createTournamentRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		payload.Name = strings.TrimSpace(payload.Name)
		payload.Game = strings.ToUpper(strings.TrimSpace(payload.Game))
		if payload.Name == "" || payload.Game == "" {
			http.Error(w, "???????? ? ???? ???????????", http.StatusBadRequest)
			return
		}
		if payload.Game != "DOTA2" && payload.Game != "CS2" {
			http.Error(w, "???? ?? ??????????????", http.StatusBadRequest)
			return
		}

		created, err := createTournament(r.Context(), conn, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := insertAuditLog(r.Context(), conn, admin.ID, created.ID, "tournament", created.ID, "create", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, created)
		realtimeHub.broadcast(created.ID, "TOURNAMENT_UPDATED")
	})

	r.Get("/api/v1/admin/tournaments/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		id := chi.URLParam(r, "id")
		item, err := getTournament(r.Context(), conn, id)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Get("/api/v1/admin/tournaments/{id}/audit", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
		if pageSize > 200 {
			pageSize = 200
		}

		result, err := listAuditLogs(r.Context(), conn, tournamentID, page, pageSize)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Patch("/api/v1/admin/tournaments/{id}", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		id := chi.URLParam(r, "id")
		var payload updateTournamentRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		updated, err := updateTournament(r.Context(), conn, id, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := insertAuditLog(r.Context(), conn, admin.ID, updated.ID, "tournament", updated.ID, "update", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, updated)
		realtimeHub.broadcast(updated.ID, "TOURNAMENT_UPDATED")
	})

	r.Delete("/api/v1/admin/tournaments/{id}", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		id := chi.URLParam(r, "id")
		deleted, err := deleteTournament(r.Context(), conn, id)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := insertAuditLog(r.Context(), conn, admin.ID, deleted.ID, "tournament", deleted.ID, "delete", nil); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		realtimeHub.broadcast(deleted.ID, "TOURNAMENT_UPDATED")
	})

	r.Get("/api/v1/admin/tournaments/{id}/teams", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 10)
		if pageSize > 500 {
			pageSize = 500
		}

		result, err := listAdminTeams(r.Context(), conn, tournamentID, page, pageSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Post("/api/v1/admin/tournaments/{id}/teams", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload createTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		created, err := createTeam(r.Context(), conn, tournamentID, payload)
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, created.TournamentID, "team", created.ID, "create", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, created)
		realtimeHub.broadcast(created.TournamentID, "TEAM_UPDATED")
	})

	r.Patch("/api/v1/admin/tournaments/{id}/teams/{teamID}", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		teamID := chi.URLParam(r, "teamID")
		var payload updateTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		updated, err := updateTeam(r.Context(), conn, tournamentID, teamID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, updated.TournamentID, "team", updated.ID, "update", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, updated)
		realtimeHub.broadcast(updated.TournamentID, "TEAM_UPDATED")
	})

	r.Patch("/api/v1/admin/tournaments/{id}/teams/{teamID}/status", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		teamID := chi.URLParam(r, "teamID")
		var payload updateTeamStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		updated, err := updateTeamStatus(r.Context(), conn, tournamentID, teamID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, updated.TournamentID, "team", updated.ID, "status_update", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, updated)
		realtimeHub.broadcast(updated.TournamentID, "TEAM_UPDATED")
	})

	r.Post("/api/v1/admin/tournaments/{id}/teams/import", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "???????????? ????? ????????", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "???? ??????????", http.StatusBadRequest)
			return
		}
		defer file.Close()

		rawDryRun := strings.TrimSpace(r.FormValue("dryRun"))
		dryRun := strings.EqualFold(rawDryRun, "true") || rawDryRun == "1"

		tournamentID := chi.URLParam(r, "id")
		result, err := importTeamsFromCSV(r.Context(), conn, tournamentID, file, dryRun)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		parsedID, _ := parseTournamentIDParam(tournamentID)
		if !dryRun && parsedID > 0 {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "tournament", parsedID, "team_import", map[string]any{
				"created":    result.Created,
				"duplicates": result.Duplicates,
				"errors":     len(result.Errors),
			}); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
		}

		writeJSON(w, http.StatusOK, result)
		if parsedID > 0 && !dryRun {
			realtimeHub.broadcast(parsedID, "TEAM_UPDATED")
		}
	})

	r.Delete("/api/v1/admin/tournaments/{id}/teams/{teamID}", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		teamID := chi.URLParam(r, "teamID")
		parsedTournamentID, parseErr := parseTournamentIDParam(tournamentID)
		if parseErr != nil {
			http.Error(w, "???????????? id ???????", http.StatusBadRequest)
			return
		}
		parsedTeamID, parseErr := strconv.ParseInt(strings.TrimSpace(teamID), 10, 64)
		if parseErr != nil || parsedTeamID < 1 {
			http.Error(w, "???????????? id ???????", http.StatusBadRequest)
			return
		}

		err = deleteTeam(r.Context(), conn, tournamentID, teamID)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := insertAuditLog(r.Context(), conn, admin.ID, parsedTournamentID, "team", parsedTeamID, "delete", map[string]string{"teamId": teamID}); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		realtimeHub.broadcast(parsedTournamentID, "TEAM_UPDATED")
	})

	r.Get("/api/v1/admin/tournaments/{id}/teams/validation", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		result, err := validateTeamsForTournament(r.Context(), conn, tournamentID)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
		realtimeHub.broadcast(result.TournamentID, "TEAM_UPDATED")
	})

	r.Post("/api/v1/admin/tournaments/{id}/teams/seeding/generate", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload generateSeedingRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		result, err := generateTeamSeeding(r.Context(), conn, tournamentID, payload.Overwrite)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		action := "seeding_generate"
		if payload.Overwrite {
			action = "seeding_regenerate"
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, result.TournamentID, "tournament", result.TournamentID, action, payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
		realtimeHub.broadcast(result.TournamentID, "TEAM_UPDATED")
	})

	r.Patch("/api/v1/admin/tournaments/{id}/teams/seeding/reorder", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload reorderSeedingRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		result, err := reorderTeamSeeding(r.Context(), conn, tournamentID, payload.TeamIDs)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, result.TournamentID, "tournament", result.TournamentID, "seeding_reorder", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
		realtimeHub.broadcast(result.TournamentID, "MATCH_UPDATED")
		realtimeHub.broadcast(result.TournamentID, "TOURNAMENT_UPDATED")
	})

	r.Post("/api/v1/admin/tournaments/{id}/bracket/generate", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload generateBracketRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		result, err := generateSingleEliminationBracket(r.Context(), conn, tournamentID, payload.Overwrite)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		action := "bracket_generate"
		if payload.Overwrite {
			action = "bracket_regenerate"
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, result.TournamentID, "tournament", result.TournamentID, action, payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Get("/api/v1/admin/tournaments/{id}/schedule", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		pageSize := parsePositiveInt(r.URL.Query().Get("pageSize"), 20)
		if pageSize > 500 {
			pageSize = 500
		}

		result, err := listScheduleItems(r.Context(), conn, tournamentID, page, pageSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Post("/api/v1/admin/tournaments/{id}/schedule/generate", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload generateScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		result, err := generateSchedule(r.Context(), conn, tournamentID, payload.Overwrite)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		action := "schedule_generate"
		if payload.Overwrite {
			action = "schedule_regenerate"
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, result.TournamentID, "tournament", result.TournamentID, action, payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Patch("/api/v1/admin/tournaments/{id}/schedule/reorder", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload reorderScheduleRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		result, err := reorderSchedule(r.Context(), conn, tournamentID, payload.MatchIDs)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, result.TournamentID, "tournament", result.TournamentID, "schedule_reorder", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	r.Get("/api/v1/admin/tournaments/{id}/round-settings", func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r); err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		items, err := listRoundSettings(r.Context(), conn, tournamentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	r.Patch("/api/v1/admin/tournaments/{id}/round-settings/{round}", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		round := parsePositiveInt(chi.URLParam(r, "round"), 0)
		if round < 1 {
			http.Error(w, "???????????? ?????", http.StatusBadRequest)
			return
		}

		var payload updateRoundSettingRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateRoundSetting(r.Context(), conn, tournamentID, round, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "round", int64(round), "update", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
		}
		writeJSON(w, http.StatusOK, item)
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
		}
	})

	r.Patch("/api/v1/admin/tournaments/{id}/matches/{matchID}/sides", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		matchID := chi.URLParam(r, "matchID")
		var payload updateMatchSidesRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateMatchSides(r.Context(), conn, tournamentID, matchID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "match", item.MatchID, "sides_update", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, item)
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
			realtimeHub.broadcast(parsedID, "SCHEDULE_UPDATED")
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Patch("/api/v1/admin/tournaments/{id}/matches/{matchID}/status", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		matchID := chi.URLParam(r, "matchID")
		var payload updateMatchStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateMatchStatus(r.Context(), conn, tournamentID, matchID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "match", item.MatchID, "status_update", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, item)
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
			realtimeHub.broadcast(parsedID, "SCHEDULE_UPDATED")
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Patch("/api/v1/admin/tournaments/{id}/matches/{matchID}/score", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		matchID := chi.URLParam(r, "matchID")
		var payload updateMatchScoreRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateMatchScore(r.Context(), conn, tournamentID, matchID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "match", item.MatchID, "score_update", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, item)
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
			realtimeHub.broadcast(parsedID, "SCHEDULE_UPDATED")
			return
		}
		writeJSON(w, http.StatusOK, item)
	})
	r.Patch("/api/v1/admin/tournaments/{id}/matches/{matchID}/result", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		matchID := chi.URLParam(r, "matchID")
		var payload updateMatchResultRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateMatchResult(r.Context(), conn, tournamentID, matchID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "match", item.MatchID, "result_update", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, item)
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
			realtimeHub.broadcast(parsedID, "SCHEDULE_UPDATED")
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Post("/api/v1/admin/tournaments/{id}/matches/{matchID}/forfeit", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		matchID := chi.URLParam(r, "matchID")
		var payload forfeitMatchRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := forfeitMatch(r.Context(), conn, tournamentID, matchID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if parsedID, parseErr := parseTournamentIDParam(tournamentID); parseErr == nil {
			if err := insertAuditLog(r.Context(), conn, admin.ID, parsedID, "match", item.MatchID, "forfeit", payload); err != nil {
				http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
				return
			}
			writeJSON(w, http.StatusOK, item)
			realtimeHub.broadcast(parsedID, "MATCH_UPDATED")
			realtimeHub.broadcast(parsedID, "SCHEDULE_UPDATED")
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	r.Post("/api/v1/admin/tournaments/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		item, err := startTournament(r.Context(), conn, tournamentID)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, item.ID, "tournament", item.ID, "start", map[string]string{"status": item.Status}); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, item)
		realtimeHub.broadcast(item.ID, "TOURNAMENT_UPDATED")
	})

	r.Patch("/api/v1/admin/tournaments/{id}/visibility", func(w http.ResponseWriter, r *http.Request) {
		admin, err := authenticateRequest(r)
		if err != nil {
			http.Error(w, "?? ????????????", http.StatusUnauthorized)
			return
		}

		tournamentID := chi.URLParam(r, "id")
		var payload updateVisibilityRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "???????????? JSON", http.StatusBadRequest)
			return
		}

		item, err := updateTournamentVisibility(r.Context(), conn, tournamentID, payload)
		if err == sql.ErrNoRows {
			http.Error(w, "?? ???????", http.StatusNotFound)
			return
		}
		var cErr clientError
		if errors.As(err, &cErr) {
			http.Error(w, cErr.Message, cErr.Status)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := insertAuditLog(r.Context(), conn, admin.ID, item.ID, "tournament", item.ID, "visibility_update", payload); err != nil {
			http.Error(w, "?? ??????? ???????? ?????-???", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, item)
		realtimeHub.broadcast(item.ID, "TOURNAMENT_UPDATED")
	})

	addr := ":" + getenvDefault("PORT", "8080")
	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

type userRecord struct {
	ID           int64
	Login        string
	PasswordHash string
}

func listTournaments(ctx context.Context, conn *sql.DB, game string) ([]tournament, error) {
	const query = `
SELECT
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at
FROM tournaments
WHERE ($1 = '' OR game = $1)
ORDER BY created_at DESC`

	rows, err := conn.QueryContext(ctx, query, game)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []tournament
	for rows.Next() {
		var item tournament
		var description sql.NullString
		var startDate sql.NullTime
		var endDate sql.NullTime
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(
			&item.ID,
			&item.Game,
			&item.Name,
			&description,
			&startDate,
			&endDate,
			&item.AllowOdd,
			&item.Status,
			&item.IsListed,
			&item.IsBracketPublished,
			&item.ScheduleVisibilityAhead,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}

		if description.Valid {
			item.Description = &description.String
		}
		if startDate.Valid {
			value := startDate.Time.Format("2006-01-02")
			item.StartDate = &value
		}
		if endDate.Valid {
			value := endDate.Time.Format("2006-01-02")
			item.EndDate = &value
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func getTournament(ctx context.Context, conn *sql.DB, id string) (tournament, error) {
	const query = `
SELECT
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at
FROM tournaments
WHERE id = $1`

	var item tournament
	var description sql.NullString
	var startDate sql.NullTime
	var endDate sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time

	err := conn.QueryRowContext(ctx, query, id).Scan(
		&item.ID,
		&item.Game,
		&item.Name,
		&description,
		&startDate,
		&endDate,
		&item.AllowOdd,
		&item.Status,
		&item.IsListed,
		&item.IsBracketPublished,
		&item.ScheduleVisibilityAhead,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return tournament{}, err
	}

	if description.Valid {
		item.Description = &description.String
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		item.StartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		item.EndDate = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	return item, nil
}

func getPublicBracket(ctx context.Context, conn *sql.DB, tournamentID string) (publicBracketResponse, error) {
	const query = `
SELECT
  m.id,
  m.round,
  m.index_in_round,
  m.status,
  m.bo,
  m.dispute_rule,
  m.team_a_id,
  ta.name,
  m.team_b_id,
  tb.name,
  m.score_a,
  m.score_b,
  m.winner_team_id
FROM matches m
LEFT JOIN teams ta ON ta.id = m.team_a_id
LEFT JOIN teams tb ON tb.id = m.team_b_id
WHERE m.tournament_id = $1
ORDER BY m.round ASC, m.index_in_round ASC`

	rows, err := conn.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return publicBracketResponse{}, err
	}
	defer rows.Close()

	rounds := make([]publicBracketRound, 0)
	roundIndex := make(map[int]int)

	for rows.Next() {
		var match publicBracketMatch
		var teamAID sql.NullInt64
		var teamBID sql.NullInt64
		var teamAName sql.NullString
		var teamBName sql.NullString
		var winnerTeamID sql.NullInt64

		if err := rows.Scan(
			&match.ID,
			&match.Round,
			&match.IndexInRound,
			&match.Status,
			&match.Bo,
			&match.DisputeRule,
			&teamAID,
			&teamAName,
			&teamBID,
			&teamBName,
			&match.ScoreA,
			&match.ScoreB,
			&winnerTeamID,
		); err != nil {
			return publicBracketResponse{}, err
		}

		if teamAID.Valid {
			v := teamAID.Int64
			match.TeamAID = &v
		}
		if teamBID.Valid {
			v := teamBID.Int64
			match.TeamBID = &v
		}
		if teamAName.Valid {
			v := teamAName.String
			match.TeamAName = &v
		}
		if teamBName.Valid {
			v := teamBName.String
			match.TeamBName = &v
		}
		if winnerTeamID.Valid {
			v := winnerTeamID.Int64
			match.WinnerTeamID = &v
		}

		index, ok := roundIndex[match.Round]
		if !ok {
			roundIndex[match.Round] = len(rounds)
			rounds = append(rounds, publicBracketRound{
				Round:   match.Round,
				Matches: []publicBracketMatch{match},
			})
		} else {
			rounds[index].Matches = append(rounds[index].Matches, match)
		}
	}
	if err := rows.Err(); err != nil {
		return publicBracketResponse{}, err
	}

	return publicBracketResponse{
		Rounds: rounds,
	}, nil
}

func getPublicSchedule(ctx context.Context, conn *sql.DB, tournament tournament) (publicScheduleResponse, error) {
	if !tournament.IsBracketPublished {
		return publicScheduleResponse{
			TournamentID: tournament.ID,
			TotalVisible: 0,
			Items:        []publicScheduleItem{},
		}, nil
	}

	const query = `
SELECT
  s.position,
  m.id,
  m.round,
  m.index_in_round,
  m.status,
  ta.name,
  tb.name,
  m.score_a,
  m.score_b,
  m.side_assignment_json
FROM schedule s
JOIN matches m ON m.id = s.match_id
LEFT JOIN teams ta ON ta.id = m.team_a_id
LEFT JOIN teams tb ON tb.id = m.team_b_id
WHERE s.tournament_id = $1
ORDER BY s.position ASC`

	rows, err := conn.QueryContext(ctx, query, tournament.ID)
	if err != nil {
		return publicScheduleResponse{}, err
	}
	defer rows.Close()

	allItems := make([]publicScheduleItem, 0)
	for rows.Next() {
		var item publicScheduleItem
		var teamAName sql.NullString
		var teamBName sql.NullString
		var sideAssignmentRaw []byte
		if err := rows.Scan(
			&item.Position,
			&item.MatchID,
			&item.Round,
			&item.IndexInRound,
			&item.Status,
			&teamAName,
			&teamBName,
			&item.ScoreA,
			&item.ScoreB,
			&sideAssignmentRaw,
		); err != nil {
			return publicScheduleResponse{}, err
		}
		if teamAName.Valid {
			value := teamAName.String
			item.TeamAName = &value
		}
		if teamBName.Valid {
			value := teamBName.String
			item.TeamBName = &value
		}

		mode, teamASide, teamBSide := normalizeSides(tournament.Game, item.IndexInRound, sideAssignmentRaw)
		item.SideMode = mode
		item.TeamASide = teamASide
		item.TeamBSide = teamBSide

		allItems = append(allItems, item)
	}
	if err := rows.Err(); err != nil {
		return publicScheduleResponse{}, err
	}

	visibleCount := resolveScheduleVisibleCount(tournament.ScheduleVisibilityAhead, len(allItems))
	if visibleCount < len(allItems) {
		allItems = allItems[:visibleCount]
	}

	return publicScheduleResponse{
		TournamentID: tournament.ID,
		TotalVisible: len(allItems),
		Items:        allItems,
	}, nil
}

func listActiveMatches(ctx context.Context, conn *sql.DB, game string, tournamentID int64) ([]activeMatchItem, error) {
	const query = `
SELECT
  m.id,
  m.tournament_id,
  t.name,
  t.game,
  m.round,
  m.index_in_round,
  m.status,
  ta.name,
  tb.name,
  m.score_a,
  m.score_b,
  m.starts_at
FROM matches m
JOIN tournaments t ON t.id = m.tournament_id
LEFT JOIN teams ta ON ta.id = m.team_a_id
LEFT JOIN teams tb ON tb.id = m.team_b_id
WHERE m.status = 'LIVE'
  AND t.is_listed = TRUE
  AND t.is_bracket_published = TRUE
  AND ($1 = '' OR t.game = $1)
  AND ($2 = 0 OR t.id = $2)
ORDER BY m.starts_at DESC NULLS LAST, m.updated_at DESC
LIMIT 100`

	rows, err := conn.QueryContext(ctx, query, game, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]activeMatchItem, 0)
	for rows.Next() {
		var item activeMatchItem
		var teamAName sql.NullString
		var teamBName sql.NullString
		var startsAt sql.NullTime
		if err := rows.Scan(
			&item.MatchID,
			&item.TournamentID,
			&item.TournamentName,
			&item.Game,
			&item.Round,
			&item.IndexInRound,
			&item.Status,
			&teamAName,
			&teamBName,
			&item.ScoreA,
			&item.ScoreB,
			&startsAt,
		); err != nil {
			return nil, err
		}
		if teamAName.Valid {
			value := teamAName.String
			item.TeamAName = &value
		}
		if teamBName.Valid {
			value := teamBName.String
			item.TeamBName = &value
		}
		if startsAt.Valid {
			value := startsAt.Time.UTC().Format(time.RFC3339)
			item.StartsAt = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type sideAssignmentPayload struct {
	Mode      string `json:"mode"`
	TeamASide string `json:"teamASide"`
	TeamBSide string `json:"teamBSide"`
}

func resolveScheduleVisibleCount(raw string, total int) int {
	value := strings.TrimSpace(strings.ToUpper(raw))
	if value == "" || value == "0" {
		return 0
	}
	if value == "ALL" {
		return total
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0
	}
	if limit > total {
		return total
	}
	return limit
}

func normalizeSides(game string, indexInRound int, raw []byte) (string, string, string) {
	firstSide, secondSide := defaultSidesForGame(game)

	payload := sideAssignmentPayload{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}

	mode := strings.ToUpper(strings.TrimSpace(payload.Mode))
	if mode == "" {
		mode = "AUTO"
	}

	if mode == "MANUAL" || mode == "RANDOM" {
		teamASide := strings.TrimSpace(payload.TeamASide)
		teamBSide := strings.TrimSpace(payload.TeamBSide)
		if teamASide == "" || teamBSide == "" {
			mode = "AUTO"
		} else {
			return mode, teamASide, teamBSide
		}
	}

	if indexInRound%2 == 1 {
		return "AUTO", firstSide, secondSide
	}
	return "AUTO", secondSide, firstSide
}

func defaultSidesForGame(game string) (string, string) {
	switch strings.ToUpper(strings.TrimSpace(game)) {
	case "CS2":
		return "CT", "T"
	default:
		return "Radiant", "Dire"
	}
}

func findUserByLogin(ctx context.Context, conn *sql.DB, login string) (userRecord, error) {
	const query = `
SELECT id, login, password_hash
FROM users
WHERE login = $1`

	var user userRecord
	err := conn.QueryRowContext(ctx, query, login).Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		return userRecord{}, err
	}
	return user, nil
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func parseTournamentIDParam(raw string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value < 1 {
		return 0, errors.New("???????????? id ???????")
	}
	return value, nil
}

type auditExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertAuditLog(
	ctx context.Context,
	exec auditExecutor,
	adminID int64,
	tournamentID int64,
	entity string,
	entityID int64,
	action string,
	payload any,
) error {
	if entity == "" || action == "" {
		return errors.New("audit requires entity and action")
	}
	raw := []byte("{}")
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		raw = encoded
	}

	const query = `
INSERT INTO audit_log (user_id, tournament_id, entity, entity_id, action, payload_json)
VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := exec.ExecContext(ctx, query, adminID, tournamentID, entity, entityID, action, raw)
	return err
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[int64]map[*websocket.Conn]struct{}),
	}
}

func (h *wsHub) serveWS(w http.ResponseWriter, r *http.Request, tournamentID int64) error {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	h.register(tournamentID, conn)
	defer h.unregister(tournamentID, conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return nil
		}
	}
}

func (h *wsHub) register(tournamentID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[tournamentID]; !ok {
		h.clients[tournamentID] = make(map[*websocket.Conn]struct{})
	}
	h.clients[tournamentID][conn] = struct{}{}
}

func (h *wsHub) unregister(tournamentID int64, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if group, ok := h.clients[tournamentID]; ok {
		delete(group, conn)
		if len(group) == 0 {
			delete(h.clients, tournamentID)
		}
	}
	_ = conn.Close()
}

func (h *wsHub) broadcast(tournamentID int64, eventType string) {
	h.mu.Lock()
	group := h.clients[tournamentID]
	connections := make([]*websocket.Conn, 0, len(group))
	for conn := range group {
		connections = append(connections, conn)
	}
	h.mu.Unlock()

	if len(connections) == 0 {
		return
	}

	event := realtimeEvent{
		Type:         eventType,
		TournamentID: tournamentID,
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	for _, conn := range connections {
		if err := conn.WriteJSON(event); err != nil {
			h.unregister(tournamentID, conn)
		}
	}
}

func listAdminTournaments(
	ctx context.Context,
	conn *sql.DB,
	game string,
	status string,
	search string,
	page int,
	pageSize int,
) (adminTournamentListResponse, error) {
	const countQuery = `
SELECT COUNT(*)
FROM tournaments
WHERE ($1 = '' OR game = $1)
  AND ($2 = '' OR status = $2)
  AND ($3 = '' OR name ILIKE '%' || $3 || '%')`

	var total int
	if err := conn.QueryRowContext(ctx, countQuery, game, status, search).Scan(&total); err != nil {
		return adminTournamentListResponse{}, err
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	const listQuery = `
SELECT
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at
FROM tournaments
WHERE ($1 = '' OR game = $1)
  AND ($2 = '' OR status = $2)
  AND ($3 = '' OR name ILIKE '%' || $3 || '%')
ORDER BY created_at DESC
LIMIT $4 OFFSET $5`

	rows, err := conn.QueryContext(ctx, listQuery, game, status, search, pageSize, offset)
	if err != nil {
		return adminTournamentListResponse{}, err
	}
	defer rows.Close()

	var items []tournament
	for rows.Next() {
		var item tournament
		var description sql.NullString
		var startDate sql.NullTime
		var endDate sql.NullTime
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(
			&item.ID,
			&item.Game,
			&item.Name,
			&description,
			&startDate,
			&endDate,
			&item.AllowOdd,
			&item.Status,
			&item.IsListed,
			&item.IsBracketPublished,
			&item.ScheduleVisibilityAhead,
			&createdAt,
			&updatedAt,
		); err != nil {
			return adminTournamentListResponse{}, err
		}

		if description.Valid {
			item.Description = &description.String
		}
		if startDate.Valid {
			value := startDate.Time.Format("2006-01-02")
			item.StartDate = &value
		}
		if endDate.Valid {
			value := endDate.Time.Format("2006-01-02")
			item.EndDate = &value
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminTournamentListResponse{}, err
	}

	return adminTournamentListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func createTournament(ctx context.Context, conn *sql.DB, payload createTournamentRequest) (tournament, error) {
	const query = `
INSERT INTO tournaments (
  game,
  name,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead
)
VALUES ($1, $2, TRUE, 'DRAFT', FALSE, FALSE, '0')
RETURNING
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at`

	var item tournament
	var description sql.NullString
	var startDate sql.NullTime
	var endDate sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time

	err := conn.QueryRowContext(ctx, query, payload.Game, payload.Name).Scan(
		&item.ID,
		&item.Game,
		&item.Name,
		&description,
		&startDate,
		&endDate,
		&item.AllowOdd,
		&item.Status,
		&item.IsListed,
		&item.IsBracketPublished,
		&item.ScheduleVisibilityAhead,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return tournament{}, err
	}

	if description.Valid {
		item.Description = &description.String
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		item.StartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		item.EndDate = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	return item, nil
}

func updateTournament(ctx context.Context, conn *sql.DB, id string, payload updateTournamentRequest) (tournament, error) {
	current, err := getTournament(ctx, conn, id)
	if err != nil {
		return tournament{}, err
	}

	nextName := current.Name
	nextGame := current.Game
	nextDescription := current.Description
	nextStartDate := current.StartDate
	nextEndDate := current.EndDate
	nextAllowOdd := current.AllowOdd

	isRunningOrFinished := current.Status == "RUNNING" || current.Status == "FINISHED"

	if payload.Name != nil {
		name := strings.TrimSpace(*payload.Name)
		if name == "" {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "название не может быть пустым"}
		}
		nextName = name
	}

	if payload.Description != nil {
		description := strings.TrimSpace(*payload.Description)
		if description == "" {
			nextDescription = nil
		} else {
			nextDescription = &description
		}
	}

	if payload.Game != nil {
		game := strings.ToUpper(strings.TrimSpace(*payload.Game))
		if game != "DOTA2" && game != "CS2" {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "игра не поддерживается"}
		}
		if isRunningOrFinished && game != current.Game {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "нельзя менять игру после запуска"}
		}
		nextGame = game
	}

	if payload.AllowOdd != nil {
		if isRunningOrFinished && *payload.AllowOdd != current.AllowOdd {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "нельзя менять настройку нечётного числа команд после запуска"}
		}
		nextAllowOdd = *payload.AllowOdd
	}

	if payload.StartDate != nil {
		normalized, err := normalizeDateString(*payload.StartDate)
		if err != nil {
			return tournament{}, err
		}
		if isRunningOrFinished && !optionalStringEqual(normalized, current.StartDate) {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "нельзя менять даты после запуска"}
		}
		nextStartDate = normalized
	}

	if payload.EndDate != nil {
		normalized, err := normalizeDateString(*payload.EndDate)
		if err != nil {
			return tournament{}, err
		}
		if isRunningOrFinished && !optionalStringEqual(normalized, current.EndDate) {
			return tournament{}, clientError{Status: http.StatusBadRequest, Message: "нельзя менять даты после запуска"}
		}
		nextEndDate = normalized
	}

	if err := validateDateRange(nextStartDate, nextEndDate); err != nil {
		return tournament{}, err
	}
	if !nextAllowOdd {
		teamCount, err := countTeamsByTournament(ctx, conn, id)
		if err != nil {
			return tournament{}, err
		}
		check := buildTeamConstraintCheck(nextAllowOdd, teamCount)
		if !check.IsValid {
			return tournament{}, clientError{
				Status:  http.StatusBadRequest,
				Message: check.MessageWithSuggestion(),
			}
		}
	}

	const query = `
UPDATE tournaments
SET
  game = $2,
  name = $3,
  description = $4,
  start_date = $5,
  end_date = $6,
  allow_odd = $7,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at`

	var descriptionArg any
	if nextDescription != nil {
		descriptionArg = *nextDescription
	}
	var startDateArg any
	if nextStartDate != nil {
		startDateArg = *nextStartDate
	}
	var endDateArg any
	if nextEndDate != nil {
		endDateArg = *nextEndDate
	}

	var item tournament
	var description sql.NullString
	var startDate sql.NullTime
	var endDate sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time

	err = conn.QueryRowContext(
		ctx,
		query,
		id,
		nextGame,
		nextName,
		descriptionArg,
		startDateArg,
		endDateArg,
		nextAllowOdd,
	).Scan(
		&item.ID,
		&item.Game,
		&item.Name,
		&description,
		&startDate,
		&endDate,
		&item.AllowOdd,
		&item.Status,
		&item.IsListed,
		&item.IsBracketPublished,
		&item.ScheduleVisibilityAhead,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return tournament{}, err
	}

	if description.Valid {
		item.Description = &description.String
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		item.StartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		item.EndDate = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	return item, nil
}

func deleteTournament(ctx context.Context, conn *sql.DB, id string) (tournament, error) {
	current, err := getTournament(ctx, conn, id)
	if err != nil {
		return tournament{}, err
	}
	if current.Status == "RUNNING" || current.Status == "FINISHED" {
		return tournament{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя удалить активный или завершённый турнир",
		}
	}
	if current.IsBracketPublished {
		return tournament{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя удалить опубликованный турнир",
		}
	}

	const deleteQuery = `
DELETE FROM tournaments
WHERE id = $1`
	result, err := conn.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return tournament{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return tournament{}, err
	}
	if rows == 0 {
		return tournament{}, sql.ErrNoRows
	}

	return current, nil
}

func normalizeDateString(raw string) (*string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, clientError{Status: http.StatusBadRequest, Message: "неверный формат даты, ожидается YYYY-MM-DD"}
	}
	formatted := parsed.Format("2006-01-02")
	return &formatted, nil
}

func validateDateRange(startDate *string, endDate *string) error {
	if startDate == nil || endDate == nil {
		return nil
	}
	start, _ := time.Parse("2006-01-02", *startDate)
	end, _ := time.Parse("2006-01-02", *endDate)
	if end.Before(start) {
		return clientError{Status: http.StatusBadRequest, Message: "дата окончания не может быть раньше даты начала"}
	}
	return nil
}

func optionalStringEqual(a *string, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func listAdminTeams(ctx context.Context, conn *sql.DB, tournamentID string, page int, pageSize int) (adminTeamListResponse, error) {
	const countQuery = `
SELECT COUNT(*)
FROM teams
WHERE tournament_id = $1`

	var total int
	if err := conn.QueryRowContext(ctx, countQuery, tournamentID).Scan(&total); err != nil {
		return adminTeamListResponse{}, err
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	const listQuery = `
SELECT
  id,
  tournament_id,
  name,
  note,
  status,
  status_reason,
  seed,
  created_at,
  updated_at
FROM teams
WHERE tournament_id = $1
ORDER BY seed NULLS LAST, created_at ASC
LIMIT $2 OFFSET $3`

	rows, err := conn.QueryContext(ctx, listQuery, tournamentID, pageSize, offset)
	if err != nil {
		return adminTeamListResponse{}, err
	}
	defer rows.Close()

	var items []team
	for rows.Next() {
		item, err := scanTeam(rows)
		if err != nil {
			return adminTeamListResponse{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return adminTeamListResponse{}, err
	}

	return adminTeamListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func createTeam(ctx context.Context, conn *sql.DB, tournamentID string, payload createTeamRequest) (team, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return team{}, clientError{Status: http.StatusBadRequest, Message: "название обязательно"}
	}
	note := normalizeOptionalString(payload.Note)

	const query = `
INSERT INTO teams (
  tournament_id,
  name,
  note,
  status
)
VALUES ($1, $2, $3, 'ACTIVE')
RETURNING
  id,
  tournament_id,
  name,
  note,
  status,
  status_reason,
  seed,
  created_at,
  updated_at`

	var noteArg any
	if note != nil {
		noteArg = *note
	}
	row := conn.QueryRowContext(ctx, query, tournamentID, name, noteArg)
	item, err := scanTeam(row)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return team{}, clientError{Status: http.StatusConflict, Message: "название команды должно быть уникальным в рамках турнира"}
		}
		return team{}, err
	}
	return item, nil
}

func updateTeam(ctx context.Context, conn *sql.DB, tournamentID string, teamID string, payload updateTeamRequest) (team, error) {
	current, err := getTeamByID(ctx, conn, tournamentID, teamID)
	if err != nil {
		return team{}, err
	}

	nextName := current.Name
	nextNote := current.Note

	if payload.Name != nil {
		name := strings.TrimSpace(*payload.Name)
		if name == "" {
			return team{}, clientError{Status: http.StatusBadRequest, Message: "название не может быть пустым"}
		}
		nextName = name
	}
	if payload.Note != nil {
		nextNote = normalizeOptionalString(payload.Note)
	}

	const query = `
UPDATE teams
SET
  name = $3,
  note = $4,
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING
  id,
  tournament_id,
  name,
  note,
  status,
  status_reason,
  seed,
  created_at,
  updated_at`

	var noteArg any
	if nextNote != nil {
		noteArg = *nextNote
	}
	row := conn.QueryRowContext(ctx, query, tournamentID, teamID, nextName, noteArg)
	item, err := scanTeam(row)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
			return team{}, clientError{Status: http.StatusConflict, Message: "название команды должно быть уникальным в рамках турнира"}
		}
		return team{}, err
	}
	return item, nil
}

func updateTeamStatus(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	teamID string,
	payload updateTeamStatusRequest,
) (team, error) {
	status := normalizeTeamStatus(payload.Status)
	if !isAllowedTeamStatus(status) {
		return team{}, clientError{Status: http.StatusBadRequest, Message: "статус команды не поддерживается"}
	}

	reason := strings.TrimSpace(payload.Reason)
	if status == "ACTIVE" {
		reason = ""
	} else {
		reason = strings.ToUpper(reason)
		if reason == "" {
			return team{}, clientError{Status: http.StatusBadRequest, Message: "для неактивного статуса нужна причина"}
		}
		if !isAllowedStatusReason(reason) {
			return team{}, clientError{Status: http.StatusBadRequest, Message: "причина статуса не поддерживается"}
		}
	}

	var reasonArg any
	if reason != "" {
		reasonArg = reason
	}

	const query = `
UPDATE teams
SET
  status = $3,
  status_reason = $4,
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING
  id,
  tournament_id,
  name,
  note,
  status,
  status_reason,
  seed,
  created_at,
  updated_at`

	row := conn.QueryRowContext(ctx, query, tournamentID, teamID, status, reasonArg)
	return scanTeam(row)
}

func importTeamsFromCSV(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	reader io.Reader,
	dryRun bool,
) (importCSVResponse, error) {
	parsedTournamentID, err := parseTournamentIDParam(tournamentID)
	if err != nil {
		return importCSVResponse{}, clientError{Status: http.StatusBadRequest, Message: "некорректный id турнира"}
	}

	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`
	if err := conn.QueryRowContext(ctx, tournamentQuery, parsedTournamentID).Scan(&parsedTournamentID); err != nil {
		return importCSVResponse{}, err
	}

	existingNames := make(map[string]struct{})
	const namesQuery = `
SELECT name
FROM teams
WHERE tournament_id = $1`
	rows, err := conn.QueryContext(ctx, namesQuery, parsedTournamentID)
	if err != nil {
		return importCSVResponse{}, err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return importCSVResponse{}, err
		}
		existingNames[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return importCSVResponse{}, err
	}
	rows.Close()

	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1

	allRows, err := csvReader.ReadAll()
	if err != nil {
		return importCSVResponse{}, clientError{Status: http.StatusBadRequest, Message: "неверный формат CSV"}
	}
	if len(allRows) == 0 {
		return importCSVResponse{}, clientError{Status: http.StatusBadRequest, Message: "CSV файл пустой"}
	}

	headerRow := allRows[0]
	hasHeader := false
	nameIndex := 0
	noteIndex := 1
	for idx, value := range headerRow {
		if strings.EqualFold(strings.TrimSpace(value), "name") {
			hasHeader = true
			nameIndex = idx
		}
		if strings.EqualFold(strings.TrimSpace(value), "note") {
			hasHeader = true
			noteIndex = idx
		}
	}

	startIndex := 0
	if hasHeader {
		startIndex = 1
	}

	type csvTeam struct {
		Name string
		Note *string
	}
	teams := make([]csvTeam, 0)
	seenNames := make(map[string]struct{})
	errorsList := make([]importCSVError, 0)
	duplicates := 0

	for idx := startIndex; idx < len(allRows); idx++ {
		row := allRows[idx]
		rowNumber := idx + 1
		rawName := ""
		rawNote := ""
		if nameIndex < len(row) {
			rawName = row[nameIndex]
		}
		if noteIndex < len(row) {
			rawNote = row[noteIndex]
		}

		name := strings.TrimSpace(rawName)
		if name == "" {
			errorsList = append(errorsList, importCSVError{Row: rowNumber, Value: rawName, Message: "название обязательно"})
			continue
		}

		nameKey := strings.ToLower(name)
		if _, ok := seenNames[nameKey]; ok {
			duplicates++
			errorsList = append(errorsList, importCSVError{Row: rowNumber, Value: name, Message: "дубликат названия в файле"})
			continue
		}
		if _, ok := existingNames[nameKey]; ok {
			duplicates++
			errorsList = append(errorsList, importCSVError{Row: rowNumber, Value: name, Message: "команда уже существует"})
			continue
		}

		seenNames[nameKey] = struct{}{}
		note := strings.TrimSpace(rawNote)
		var notePtr *string
		if note != "" {
			notePtr = &note
		}
		teams = append(teams, csvTeam{Name: name, Note: notePtr})
	}

	response := importCSVResponse{
		Mode:       "dryRun",
		Total:      len(allRows) - startIndex,
		Created:    len(teams),
		Duplicates: duplicates,
		Errors:     errorsList,
	}
	if dryRun {
		return response, nil
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return importCSVResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	created := 0
	for _, item := range teams {
		var noteArg any
		if item.Note != nil {
			noteArg = *item.Note
		}
		const insertQuery = `
INSERT INTO teams (tournament_id, name, note, status)
VALUES ($1, $2, $3, 'ACTIVE')`
		if _, err := tx.ExecContext(ctx, insertQuery, parsedTournamentID, item.Name, noteArg); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key value") {
				duplicates++
				continue
			}
			return importCSVResponse{}, err
		}
		created++
	}

	if err := tx.Commit(); err != nil {
		return importCSVResponse{}, err
	}

	response.Mode = "apply"
	response.Created = created
	response.Duplicates = duplicates
	return response, nil
}

func deleteTeam(ctx context.Context, conn *sql.DB, tournamentID string, teamID string) error {
	const query = `
DELETE FROM teams
WHERE tournament_id = $1 AND id = $2`

	result, err := conn.ExecContext(ctx, query, tournamentID, teamID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func getTeamByID(ctx context.Context, conn *sql.DB, tournamentID string, teamID string) (team, error) {
	const query = `
SELECT
  id,
  tournament_id,
  name,
  note,
  status,
  status_reason,
  seed,
  created_at,
  updated_at
FROM teams
WHERE tournament_id = $1 AND id = $2`

	row := conn.QueryRowContext(ctx, query, tournamentID, teamID)
	return scanTeam(row)
}

type teamScanner interface {
	Scan(dest ...any) error
}

func scanTeam(scanner teamScanner) (team, error) {
	var item team
	var note sql.NullString
	var statusReason sql.NullString
	var seed sql.NullInt32
	var createdAt time.Time
	var updatedAt time.Time
	err := scanner.Scan(
		&item.ID,
		&item.TournamentID,
		&item.Name,
		&note,
		&item.Status,
		&statusReason,
		&seed,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return team{}, err
	}
	if note.Valid {
		item.Note = &note.String
	}
	if statusReason.Valid {
		item.StatusReason = &statusReason.String
	}
	if seed.Valid {
		value := int(seed.Int32)
		item.Seed = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return item, nil
}

func normalizeOptionalString(input *string) *string {
	if input == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*input)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func generateTeamSeeding(ctx context.Context, conn *sql.DB, tournamentID string, overwrite bool) (generateSeedingResponse, error) {
	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`

	var parsedTournamentID int64
	if err := conn.QueryRowContext(ctx, tournamentQuery, tournamentID).Scan(&parsedTournamentID); err != nil {
		return generateSeedingResponse{}, err
	}

	const teamsQuery = `
SELECT id, seed
FROM teams
WHERE tournament_id = $1
ORDER BY id ASC`
	rows, err := conn.QueryContext(ctx, teamsQuery, tournamentID)
	if err != nil {
		return generateSeedingResponse{}, err
	}
	defer rows.Close()

	type teamSeedRow struct {
		ID   int64
		Seed sql.NullInt32
	}
	var source []teamSeedRow
	for rows.Next() {
		var row teamSeedRow
		if err := rows.Scan(&row.ID, &row.Seed); err != nil {
			return generateSeedingResponse{}, err
		}
		source = append(source, row)
	}
	if err := rows.Err(); err != nil {
		return generateSeedingResponse{}, err
	}

	if len(source) < 2 {
		return generateSeedingResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нужно минимум 2 команды, чтобы сформировать порядок",
		}
	}

	if !overwrite {
		for _, row := range source {
			if row.Seed.Valid {
				return generateSeedingResponse{}, clientError{
					Status:  http.StatusConflict,
					Message: "порядок уже сформирован, используйте перемешивание",
				}
			}
		}
	}

	ids := make([]int64, 0, len(source))
	for _, row := range source {
		ids = append(ids, row.ID)
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd.Shuffle(len(ids), func(i int, j int) {
		ids[i], ids[j] = ids[j], ids[i]
	})

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return generateSeedingResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const updateQuery = `
UPDATE teams
SET seed = $3, updated_at = NOW()
WHERE tournament_id = $1 AND id = $2`
	for index, teamID := range ids {
		if _, err := tx.ExecContext(ctx, updateQuery, tournamentID, teamID, index+1); err != nil {
			return generateSeedingResponse{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return generateSeedingResponse{}, err
	}

	return generateSeedingResponse{
		TournamentID: parsedTournamentID,
		TeamCount:    len(ids),
		UpdatedCount: len(ids),
	}, nil
}

func reorderTeamSeeding(ctx context.Context, conn *sql.DB, tournamentID string, teamIDs []int64) (generateSeedingResponse, error) {
	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`

	var parsedTournamentID int64
	if err := conn.QueryRowContext(ctx, tournamentQuery, tournamentID).Scan(&parsedTournamentID); err != nil {
		return generateSeedingResponse{}, err
	}

	const teamsQuery = `
SELECT id
FROM teams
WHERE tournament_id = $1`
	rows, err := conn.QueryContext(ctx, teamsQuery, tournamentID)
	if err != nil {
		return generateSeedingResponse{}, err
	}
	defer rows.Close()

	existingIDs := make([]int64, 0)
	existingSet := make(map[int64]struct{})
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return generateSeedingResponse{}, err
		}
		existingIDs = append(existingIDs, id)
		existingSet[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return generateSeedingResponse{}, err
	}

	if len(existingIDs) < 2 {
		return generateSeedingResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нужно минимум 2 команды, чтобы сформировать порядок",
		}
	}
	if len(teamIDs) != len(existingIDs) {
		return generateSeedingResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "список команд должен содержать все команды турнира ровно по одному разу",
		}
	}

	seen := make(map[int64]struct{}, len(teamIDs))
	for _, id := range teamIDs {
		if _, ok := existingSet[id]; !ok {
			return generateSeedingResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "в списке есть команды, не относящиеся к турниру",
			}
		}
		if _, dup := seen[id]; dup {
			return generateSeedingResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "в списке есть дубликаты команд",
			}
		}
		seen[id] = struct{}{}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return generateSeedingResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const updateQuery = `
UPDATE teams
SET seed = $3, updated_at = NOW()
WHERE tournament_id = $1 AND id = $2`
	for index, teamID := range teamIDs {
		if _, err := tx.ExecContext(ctx, updateQuery, tournamentID, teamID, index+1); err != nil {
			return generateSeedingResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return generateSeedingResponse{}, err
	}

	return generateSeedingResponse{
		TournamentID: parsedTournamentID,
		TeamCount:    len(teamIDs),
		UpdatedCount: len(teamIDs),
	}, nil
}

func generateSingleEliminationBracket(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	overwrite bool,
) (generateBracketResponse, error) {
	const tournamentQuery = `
SELECT id, status, allow_odd
FROM tournaments
WHERE id = $1`

	var parsedTournamentID int64
	var status string
	var allowOdd bool
	if err := conn.QueryRowContext(ctx, tournamentQuery, tournamentID).Scan(&parsedTournamentID, &status, &allowOdd); err != nil {
		return generateBracketResponse{}, err
	}

	if status == "RUNNING" || status == "FINISHED" {
		return generateBracketResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "генерация сетки доступна только для статусов Черновик/Готов (DRAFT/READY)",
		}
	}

	const existingQuery = `
SELECT COUNT(*)
FROM matches
WHERE tournament_id = $1`
	var existingCount int
	if err := conn.QueryRowContext(ctx, existingQuery, tournamentID).Scan(&existingCount); err != nil {
		return generateBracketResponse{}, err
	}
	if existingCount > 0 && !overwrite {
		return generateBracketResponse{}, clientError{
			Status:  http.StatusConflict,
			Message: "матчи уже существуют; используйте параметр overwrite=true для пересоздания сетки",
		}
	}

	const teamsQuery = `
SELECT id
FROM teams
WHERE tournament_id = $1 AND status = 'ACTIVE'
ORDER BY seed NULLS LAST, created_at ASC`
	rows, err := conn.QueryContext(ctx, teamsQuery, tournamentID)
	if err != nil {
		return generateBracketResponse{}, err
	}
	defer rows.Close()

	teamIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return generateBracketResponse{}, err
		}
		teamIDs = append(teamIDs, id)
	}
	if err := rows.Err(); err != nil {
		return generateBracketResponse{}, err
	}

	check := buildTeamConstraintCheck(allowOdd, len(teamIDs))
	if !check.IsValid {
		return generateBracketResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: check.MessageWithSuggestion(),
		}
	}

	bracketSize := nextPowerOfTwo(len(teamIDs))
	byeSlots := bracketSize - len(teamIDs)
	roundsCount := 0
	for matches := bracketSize / 2; matches >= 1; matches /= 2 {
		roundsCount++
	}

	slots := make([]*int64, bracketSize)
	for index, teamID := range teamIDs {
		value := teamID
		slots[index] = &value
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return generateBracketResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if overwrite {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schedule WHERE tournament_id = $1`, tournamentID); err != nil {
			return generateBracketResponse{}, err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM matches WHERE tournament_id = $1`, tournamentID); err != nil {
			return generateBracketResponse{}, err
		}
	}

	roundInfos := make([]bracketRoundInfo, 0, roundsCount)
	totalMatches := 0
	for round := 1; round <= roundsCount; round++ {
		matchesInRound := bracketSize >> round
		totalMatches += matchesInRound
		roundInfos = append(roundInfos, bracketRoundInfo{
			Round:          round,
			MatchesInRound: matchesInRound,
		})

		for indexInRound := 1; indexInRound <= matchesInRound; indexInRound++ {
			var teamAArg any
			var teamBArg any
			if round == 1 {
				leftIndex := indexInRound - 1
				rightIndex := bracketSize - indexInRound
				if slots[leftIndex] != nil {
					teamAArg = *slots[leftIndex]
				}
				if slots[rightIndex] != nil {
					teamBArg = *slots[rightIndex]
				}
			}

			const insertMatchQuery = `
INSERT INTO matches (
  tournament_id,
  round,
  index_in_round,
  status,
  bo,
  dispute_rule,
  team_a_id,
  team_b_id,
  score_a,
  score_b,
  side_assignment_json
)
VALUES ($1, $2, $3, 'SCHEDULED', 1, 'ADMIN_DECISION', $4, $5, 0, 0, '{}'::jsonb)`
			if _, err := tx.ExecContext(ctx, insertMatchQuery, tournamentID, round, indexInRound, teamAArg, teamBArg); err != nil {
				return generateBracketResponse{}, err
			}
		}
	}

	if err := autoAdvanceByesInTx(ctx, tx, parsedTournamentID); err != nil {
		return generateBracketResponse{}, err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE tournaments SET status = 'READY', updated_at = NOW() WHERE id = $1`, tournamentID); err != nil {
		return generateBracketResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return generateBracketResponse{}, err
	}

	return generateBracketResponse{
		TournamentID:  parsedTournamentID,
		BracketSize:   bracketSize,
		RoundsCount:   roundsCount,
		TotalMatches:  totalMatches,
		ByeSlots:      byeSlots,
		TournamentNow: "READY",
		Rounds:        roundInfos,
	}, nil
}

func listScheduleItems(ctx context.Context, conn *sql.DB, tournamentID string, page int, pageSize int) (scheduleListResponse, error) {
	tournament, err := getTournament(ctx, conn, tournamentID)
	if err != nil {
		return scheduleListResponse{}, err
	}

	const countQuery = `
SELECT COUNT(*)
FROM schedule
WHERE tournament_id = $1`

	var total int
	if err := conn.QueryRowContext(ctx, countQuery, tournamentID).Scan(&total); err != nil {
		return scheduleListResponse{}, err
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	const listQuery = `
SELECT
  s.id,
  s.tournament_id,
  s.match_id,
  s.position,
  m.round,
  m.index_in_round,
  m.status,
  m.bo,
  m.team_a_id,
  m.team_b_id,
  ta.name,
  tb.name,
  m.score_a,
  m.score_b,
  m.winner_team_id,
  m.side_assignment_json
FROM schedule s
JOIN matches m ON m.id = s.match_id
LEFT JOIN teams ta ON ta.id = m.team_a_id
LEFT JOIN teams tb ON tb.id = m.team_b_id
WHERE s.tournament_id = $1
ORDER BY s.position ASC
LIMIT $2 OFFSET $3`

	rows, err := conn.QueryContext(ctx, listQuery, tournamentID, pageSize, offset)
	if err != nil {
		return scheduleListResponse{}, err
	}
	defer rows.Close()

	items := make([]scheduleItem, 0)
	for rows.Next() {
		var item scheduleItem
		var teamAID sql.NullInt64
		var teamBID sql.NullInt64
		var teamAName sql.NullString
		var teamBName sql.NullString
		var winnerTeamID sql.NullInt64
		var sideAssignmentRaw []byte
		if err := rows.Scan(
			&item.ID,
			&item.TournamentID,
			&item.MatchID,
			&item.Position,
			&item.MatchRound,
			&item.MatchIndex,
			&item.Status,
			&item.Bo,
			&teamAID,
			&teamBID,
			&teamAName,
			&teamBName,
			&item.ScoreA,
			&item.ScoreB,
			&winnerTeamID,
			&sideAssignmentRaw,
		); err != nil {
			return scheduleListResponse{}, err
		}
		if teamAID.Valid {
			value := teamAID.Int64
			item.TeamAID = &value
		}
		if teamBID.Valid {
			value := teamBID.Int64
			item.TeamBID = &value
		}
		if teamAName.Valid {
			value := teamAName.String
			item.TeamAName = &value
		}
		if teamBName.Valid {
			value := teamBName.String
			item.TeamBName = &value
		}
		if winnerTeamID.Valid {
			value := winnerTeamID.Int64
			item.WinnerTeamID = &value
		}
		mode, teamASide, teamBSide := normalizeSides(tournament.Game, item.MatchIndex, sideAssignmentRaw)
		item.SideMode = mode
		item.TeamASide = teamASide
		item.TeamBSide = teamBSide
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return scheduleListResponse{}, err
	}

	return scheduleListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func listAuditLogs(ctx context.Context, conn *sql.DB, tournamentID string, page int, pageSize int) (auditLogListResponse, error) {
	parsedID, err := parseTournamentIDParam(tournamentID)
	if err != nil {
		return auditLogListResponse{}, err
	}

	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`
	var exists int64
	if err := conn.QueryRowContext(ctx, tournamentQuery, parsedID).Scan(&exists); err != nil {
		return auditLogListResponse{}, err
	}

	const countQuery = `
SELECT COUNT(*)
FROM audit_log
WHERE tournament_id = $1`
	var total int
	if err := conn.QueryRowContext(ctx, countQuery, parsedID).Scan(&total); err != nil {
		return auditLogListResponse{}, err
	}

	totalPages := 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	const listQuery = `
SELECT
  a.id,
  a.user_id,
  u.login,
  a.entity,
  a.entity_id,
  a.action,
  a.payload_json,
  a.created_at
FROM audit_log a
LEFT JOIN users u ON u.id = a.user_id
WHERE a.tournament_id = $1
ORDER BY a.created_at DESC
LIMIT $2 OFFSET $3`

	rows, err := conn.QueryContext(ctx, listQuery, parsedID, pageSize, offset)
	if err != nil {
		return auditLogListResponse{}, err
	}
	defer rows.Close()

	items := make([]auditLogEntry, 0)
	for rows.Next() {
		var item auditLogEntry
		var userID sql.NullInt64
		var login sql.NullString
		var payload []byte
		var createdAt time.Time
		if err := rows.Scan(
			&item.ID,
			&userID,
			&login,
			&item.Entity,
			&item.EntityID,
			&item.Action,
			&payload,
			&createdAt,
		); err != nil {
			return auditLogListResponse{}, err
		}
		item.TournamentID = parsedID
		if userID.Valid {
			value := userID.Int64
			item.UserID = &value
		}
		if login.Valid {
			value := login.String
			item.UserLogin = &value
		}
		if len(payload) > 0 {
			item.Payload = payload
		} else {
			item.Payload = []byte("{}")
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return auditLogListResponse{}, err
	}

	return auditLogListResponse{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func generateSchedule(ctx context.Context, conn *sql.DB, tournamentID string, overwrite bool) (generateScheduleResponse, error) {
	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`
	var parsedTournamentID int64
	if err := conn.QueryRowContext(ctx, tournamentQuery, tournamentID).Scan(&parsedTournamentID); err != nil {
		return generateScheduleResponse{}, err
	}

	const scheduleCountQuery = `
SELECT COUNT(*)
FROM schedule
WHERE tournament_id = $1`
	var existingScheduleCount int
	if err := conn.QueryRowContext(ctx, scheduleCountQuery, tournamentID).Scan(&existingScheduleCount); err != nil {
		return generateScheduleResponse{}, err
	}
	if existingScheduleCount > 0 && !overwrite {
		return generateScheduleResponse{}, clientError{
			Status:  http.StatusConflict,
			Message: "расписание уже существует; используйте параметр overwrite=true для пересоздания",
		}
	}

	const matchesQuery = `
SELECT id
FROM matches
WHERE tournament_id = $1
ORDER BY round ASC, index_in_round ASC`
	rows, err := conn.QueryContext(ctx, matchesQuery, tournamentID)
	if err != nil {
		return generateScheduleResponse{}, err
	}
	defer rows.Close()

	matchIDs := make([]int64, 0)
	for rows.Next() {
		var matchID int64
		if err := rows.Scan(&matchID); err != nil {
			return generateScheduleResponse{}, err
		}
		matchIDs = append(matchIDs, matchID)
	}
	if err := rows.Err(); err != nil {
		return generateScheduleResponse{}, err
	}
	if len(matchIDs) == 0 {
		return generateScheduleResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "матчи не найдены; сначала сгенерируйте сетку",
		}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return generateScheduleResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if overwrite {
		if _, err := tx.ExecContext(ctx, `DELETE FROM schedule WHERE tournament_id = $1`, tournamentID); err != nil {
			return generateScheduleResponse{}, err
		}
	}

	const insertQuery = `
INSERT INTO schedule (tournament_id, match_id, position)
VALUES ($1, $2, $3)`
	for index, matchID := range matchIDs {
		if _, err := tx.ExecContext(ctx, insertQuery, tournamentID, matchID, index+1); err != nil {
			return generateScheduleResponse{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return generateScheduleResponse{}, err
	}

	return generateScheduleResponse{
		TournamentID: parsedTournamentID,
		TotalMatches: len(matchIDs),
		UpdatedCount: len(matchIDs),
	}, nil
}

func reorderSchedule(ctx context.Context, conn *sql.DB, tournamentID string, matchIDs []int64) (generateScheduleResponse, error) {
	const tournamentQuery = `
SELECT id
FROM tournaments
WHERE id = $1`
	var parsedTournamentID int64
	if err := conn.QueryRowContext(ctx, tournamentQuery, tournamentID).Scan(&parsedTournamentID); err != nil {
		return generateScheduleResponse{}, err
	}

	const existingQuery = `
SELECT match_id
FROM schedule
WHERE tournament_id = $1`
	rows, err := conn.QueryContext(ctx, existingQuery, tournamentID)
	if err != nil {
		return generateScheduleResponse{}, err
	}
	defer rows.Close()

	existingSet := make(map[int64]struct{})
	existingCount := 0
	for rows.Next() {
		var matchID int64
		if err := rows.Scan(&matchID); err != nil {
			return generateScheduleResponse{}, err
		}
		existingSet[matchID] = struct{}{}
		existingCount++
	}
	if err := rows.Err(); err != nil {
		return generateScheduleResponse{}, err
	}
	if existingCount == 0 {
		return generateScheduleResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "расписание пустое; сначала сгенерируйте расписание",
		}
	}
	if len(matchIDs) != existingCount {
		return generateScheduleResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "список матчей должен содержать все матчи расписания ровно по одному разу",
		}
	}

	seen := make(map[int64]struct{}, len(matchIDs))
	for _, id := range matchIDs {
		if _, ok := existingSet[id]; !ok {
			return generateScheduleResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "в списке есть матч вне расписания",
			}
		}
		if _, dup := seen[id]; dup {
			return generateScheduleResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "в списке есть дубликаты матчей",
			}
		}
		seen[id] = struct{}{}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return generateScheduleResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const updateQuery = `
UPDATE schedule
SET position = $3
WHERE tournament_id = $1 AND match_id = $2`
	for index, matchID := range matchIDs {
		if _, err := tx.ExecContext(ctx, updateQuery, tournamentID, matchID, -(index + 1)); err != nil {
			return generateScheduleResponse{}, err
		}
	}
	for index, matchID := range matchIDs {
		if _, err := tx.ExecContext(ctx, updateQuery, tournamentID, matchID, index+1); err != nil {
			return generateScheduleResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return generateScheduleResponse{}, err
	}

	return generateScheduleResponse{
		TournamentID: parsedTournamentID,
		TotalMatches: existingCount,
		UpdatedCount: existingCount,
	}, nil
}

func listRoundSettings(ctx context.Context, conn *sql.DB, tournamentID string) ([]roundSetting, error) {
	const query = `
SELECT
  round,
  MIN(bo) AS bo,
  MIN(dispute_rule) AS dispute_rule,
  COUNT(*) AS matches_count
FROM matches
WHERE tournament_id = $1
GROUP BY round
ORDER BY round ASC`

	rows, err := conn.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]roundSetting, 0)
	for rows.Next() {
		var item roundSetting
		if err := rows.Scan(&item.Round, &item.Bo, &item.DisputeRule, &item.Matches); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func updateRoundSetting(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	round int,
	payload updateRoundSettingRequest,
) (roundSetting, error) {
	if payload.Bo < 1 || payload.Bo%2 == 0 {
		return roundSetting{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "формат серии (BO) должен быть положительным нечётным числом",
		}
	}

	rule := normalizeDisputeRule(payload.DisputeRule)
	if !isAllowedDisputeRule(rule) {
		return roundSetting{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "правило спора не поддерживается",
		}
	}

	const updateQuery = `
UPDATE matches
SET bo = $3, dispute_rule = $4, updated_at = NOW()
WHERE tournament_id = $1 AND round = $2`
	result, err := conn.ExecContext(ctx, updateQuery, tournamentID, round, payload.Bo, rule)
	if err != nil {
		return roundSetting{}, err
	}
	updatedCount, _ := result.RowsAffected()
	if updatedCount == 0 {
		return roundSetting{}, sql.ErrNoRows
	}

	return roundSetting{
		Round:       round,
		Bo:          payload.Bo,
		DisputeRule: rule,
		Matches:     int(updatedCount),
	}, nil
}

func normalizeDisputeRule(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isAllowedDisputeRule(value string) bool {
	switch value {
	case "OVERTIME", "REPLAY", "ADMIN_DECISION", "COIN_TOSS", "FORFEIT":
		return true
	default:
		return false
	}
}

func normalizeTeamStatus(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func isAllowedTeamStatus(value string) bool {
	switch value {
	case "ACTIVE", "WITHDRAWN", "DISQUALIFIED":
		return true
	default:
		return false
	}
}

func isAllowedStatusReason(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CHEATING", "TOXIC_BEHAVIOR", "NO_SHOW", "TECH_ISSUES", "OTHER":
		return true
	default:
		return false
	}
}

func updateMatchSides(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	matchID string,
	payload updateMatchSidesRequest,
) (scheduleItem, error) {
	tournament, err := getTournament(ctx, conn, tournamentID)
	if err != nil {
		return scheduleItem{}, err
	}

	const matchQuery = `
SELECT
  m.id,
  m.index_in_round,
  m.status,
  ta.name,
  tb.name
FROM matches m
LEFT JOIN teams ta ON ta.id = m.team_a_id
LEFT JOIN teams tb ON tb.id = m.team_b_id
WHERE m.tournament_id = $1 AND m.id = $2`

	var item scheduleItem
	var teamAName sql.NullString
	var teamBName sql.NullString
	if err := conn.QueryRowContext(ctx, matchQuery, tournamentID, matchID).Scan(
		&item.MatchID,
		&item.MatchIndex,
		&item.Status,
		&teamAName,
		&teamBName,
	); err != nil {
		return scheduleItem{}, err
	}
	if teamAName.Valid {
		value := teamAName.String
		item.TeamAName = &value
	}
	if teamBName.Valid {
		value := teamBName.String
		item.TeamBName = &value
	}

	mode := strings.ToUpper(strings.TrimSpace(payload.Mode))
	if mode == "" {
		mode = "RANDOM"
	}

	firstSide, secondSide := defaultSidesForGame(tournament.Game)
	teamASide := firstSide
	teamBSide := secondSide

	if mode == "RANDOM" {
		if rand.Intn(2) == 1 {
			teamASide, teamBSide = teamBSide, teamASide
		}
	} else if mode == "MANUAL" {
		teamASide = strings.TrimSpace(payload.TeamASide)
		teamBSide = strings.TrimSpace(payload.TeamBSide)
		if !isValidSidesForGame(tournament.Game, teamASide, teamBSide) {
			return scheduleItem{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "некорректные стороны для выбранной игры",
			}
		}
		if teamASide == teamBSide {
			return scheduleItem{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "стороны команд A и B должны отличаться",
			}
		}
	} else {
		return scheduleItem{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "режим должен быть Случайный или Ручной (RANDOM/MANUAL)",
		}
	}

	assignment := sideAssignmentPayload{
		Mode:      mode,
		TeamASide: teamASide,
		TeamBSide: teamBSide,
	}
	raw, err := json.Marshal(assignment)
	if err != nil {
		return scheduleItem{}, err
	}

	const updateQuery = `
UPDATE matches
SET side_assignment_json = $3, updated_at = NOW()
WHERE tournament_id = $1 AND id = $2`
	result, err := conn.ExecContext(ctx, updateQuery, tournamentID, matchID, raw)
	if err != nil {
		return scheduleItem{}, err
	}
	updatedRows, _ := result.RowsAffected()
	if updatedRows == 0 {
		return scheduleItem{}, sql.ErrNoRows
	}

	item.TournamentID = tournament.ID
	item.SideMode = mode
	item.TeamASide = teamASide
	item.TeamBSide = teamBSide
	return item, nil
}

func updateMatchStatus(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	matchID string,
	payload updateMatchStatusRequest,
) (matchStatusResponse, error) {
	action := strings.ToUpper(strings.TrimSpace(payload.Action))
	if action == "" {
		return matchStatusResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "действие обязательно",
		}
	}

	const query = `
SELECT
  t.id,
  t.status,
  m.id,
  m.status
FROM matches m
JOIN tournaments t ON t.id = m.tournament_id
WHERE m.tournament_id = $1 AND m.id = $2`

	var tournamentIDParsed int64
	var tournamentStatus string
	var matchIDParsed int64
	var matchStatus string
	if err := conn.QueryRowContext(ctx, query, tournamentID, matchID).Scan(
		&tournamentIDParsed,
		&tournamentStatus,
		&matchIDParsed,
		&matchStatus,
	); err != nil {
		return matchStatusResponse{}, err
	}

	if tournamentStatus != "RUNNING" {
		return matchStatusResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "турнир должен быть в статусе Идёт (RUNNING) для управления матчами",
		}
	}

	setStart := false
	nextStatus := ""

	switch action {
	case "START":
		if matchStatus == "LIVE" {
			return matchStatusResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "матч уже в статусе LIVE (идёт)",
			}
		}
		if matchStatus == "PAUSED" {
			return matchStatusResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "матч на паузе; используйте RESUME (возобновить)",
			}
		}
		if matchStatus == "FINISHED" || matchStatus == "CANCELED" {
			return matchStatusResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "нельзя запускать завершённый или отменённый матч",
			}
		}
		nextStatus = "LIVE"
		setStart = true
	case "PAUSE":
		if matchStatus != "LIVE" {
			return matchStatusResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "матч должен быть в статусе LIVE (идёт) для паузы",
			}
		}
		nextStatus = "PAUSED"
	case "RESUME":
		if matchStatus != "PAUSED" {
			return matchStatusResponse{}, clientError{
				Status:  http.StatusBadRequest,
				Message: "матч должен быть в статусе PAUSED (пауза) для возобновления",
			}
		}
		nextStatus = "LIVE"
		setStart = true
	default:
		return matchStatusResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "действие должно быть START, PAUSE или RESUME",
		}
	}

	const updateQuery = `
UPDATE matches
SET
  status = $3,
  starts_at = CASE WHEN $4 THEN COALESCE(starts_at, NOW()) ELSE starts_at END,
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING status, starts_at, updated_at`

	var updatedStatus string
	var updatedStartsAt sql.NullTime
	var updatedAt time.Time
	if err := conn.QueryRowContext(ctx, updateQuery, tournamentID, matchID, nextStatus, setStart).Scan(
		&updatedStatus,
		&updatedStartsAt,
		&updatedAt,
	); err != nil {
		return matchStatusResponse{}, err
	}

	response := matchStatusResponse{
		TournamentID: tournamentIDParsed,
		MatchID:      matchIDParsed,
		Status:       updatedStatus,
		UpdatedAt:    updatedAt.UTC().Format(time.RFC3339),
	}
	if updatedStartsAt.Valid {
		value := updatedStartsAt.Time.UTC().Format(time.RFC3339)
		response.StartsAt = &value
	}
	return response, nil
}

func updateMatchScore(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	matchID string,
	payload updateMatchScoreRequest,
) (matchScoreResponse, error) {
	if payload.ScoreA < 0 || payload.ScoreB < 0 {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "???? ?? ????? ???? ??????????????",
		}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return matchScoreResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const query = `
SELECT
  t.id,
  t.status,
  m.id,
  m.status,
  m.bo,
  m.team_a_id,
  m.team_b_id
FROM matches m
JOIN tournaments t ON t.id = m.tournament_id
WHERE m.tournament_id = $1 AND m.id = $2`

	var tournamentIDParsed int64
	var tournamentStatus string
	var matchIDParsed int64
	var matchStatus string
	var bestOf int
	var teamAID sql.NullInt64
	var teamBID sql.NullInt64
	if err := tx.QueryRowContext(ctx, query, tournamentID, matchID).Scan(
		&tournamentIDParsed,
		&tournamentStatus,
		&matchIDParsed,
		&matchStatus,
		&bestOf,
		&teamAID,
		&teamBID,
	); err != nil {
		return matchScoreResponse{}, err
	}

	if tournamentStatus != "RUNNING" {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "?????? ?????? ???? ? ??????? ???? (RUNNING) ??? ???????? ?????",
		}
	}
	if matchStatus == "FINISHED" {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "???? ??? ????????",
		}
	}
	if matchStatus == "CANCELED" {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "?????? ????????? ?????????? ????",
		}
	}
	if bestOf < 1 || bestOf%2 == 0 {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "???????????? ?????? ????? (BO) ?????",
		}
	}
	if !teamAID.Valid || !teamBID.Valid {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "? ????? ?????? ???? ????????? ??? ???????",
		}
	}

	requiredWins := bestOf/2 + 1
	if payload.ScoreA > requiredWins || payload.ScoreB > requiredWins {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "???? ?? ????? ????????? ????????? ????? ?????",
		}
	}
	if payload.ScoreA == payload.ScoreB && payload.ScoreA == requiredWins {
		return matchScoreResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "???????? ???? ?????????? ??? ?????????? ?????",
		}
	}

	const updateQuery = `
UPDATE matches
SET
  score_a = $3,
  score_b = $4,
  winner_team_id = NULL,
  status = 'PAUSED',
  ended_at = NULL,
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING status, updated_at`

	var updatedStatus string
	var updatedAt time.Time
	if err := tx.QueryRowContext(ctx, updateQuery, tournamentID, matchID, payload.ScoreA, payload.ScoreB).Scan(
		&updatedStatus,
		&updatedAt,
	); err != nil {
		return matchScoreResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return matchScoreResponse{}, err
	}

	return matchScoreResponse{
		TournamentID: tournamentIDParsed,
		MatchID:      matchIDParsed,
		Status:       updatedStatus,
		ScoreA:       payload.ScoreA,
		ScoreB:       payload.ScoreB,
		UpdatedAt:    updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func updateMatchResult(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	matchID string,
	payload updateMatchResultRequest,
) (matchResultResponse, error) {
	if payload.ScoreA < 0 || payload.ScoreB < 0 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "очки не могут быть отрицательными",
		}
	}
	if payload.WinnerTeamID == 0 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нужно указать победившую команду",
		}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return matchResultResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const query = `
SELECT
  t.id,
  t.status,
  m.id,
  m.status,
  m.round,
  m.index_in_round,
  m.bo,
  m.team_a_id,
  m.team_b_id
FROM matches m
JOIN tournaments t ON t.id = m.tournament_id
WHERE m.tournament_id = $1 AND m.id = $2`

	var tournamentIDParsed int64
	var tournamentStatus string
	var matchIDParsed int64
	var matchStatus string
	var matchRound int
	var matchIndex int
	var bestOf int
	var teamAID sql.NullInt64
	var teamBID sql.NullInt64
	if err := tx.QueryRowContext(ctx, query, tournamentID, matchID).Scan(
		&tournamentIDParsed,
		&tournamentStatus,
		&matchIDParsed,
		&matchStatus,
		&matchRound,
		&matchIndex,
		&bestOf,
		&teamAID,
		&teamBID,
	); err != nil {
		return matchResultResponse{}, err
	}

	if tournamentStatus != "RUNNING" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "турнир должен быть в статусе Идёт (RUNNING) для завершения матчей",
		}
	}
	if matchStatus == "FINISHED" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "матч уже завершён",
		}
	}
	if matchStatus == "CANCELED" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя обновлять отменённый матч",
		}
	}
	if bestOf < 1 || bestOf%2 == 0 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "некорректный формат серии (BO) матча",
		}
	}
	if !teamAID.Valid || !teamBID.Valid {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "в матче должны быть назначены обе команды",
		}
	}

	requiredWins := bestOf/2 + 1
	maxScore := payload.ScoreA
	if payload.ScoreB > maxScore {
		maxScore = payload.ScoreB
	}
	if maxScore != requiredWins {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "счёт не соответствует требуемому числу побед для формата BO",
		}
	}
	if payload.ScoreA == payload.ScoreB {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "счёт не может быть равным",
		}
	}

	var winnerExpected int64
	if payload.ScoreA > payload.ScoreB {
		winnerExpected = teamAID.Int64
	} else {
		winnerExpected = teamBID.Int64
	}
	if payload.WinnerTeamID != winnerExpected {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "победившая команда не соответствует победителю по счёту",
		}
	}

	const updateQuery = `
UPDATE matches
SET
  score_a = $3,
  score_b = $4,
  winner_team_id = $5,
  status = 'FINISHED',
  ended_at = NOW(),
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING status, ended_at, updated_at`

	var updatedStatus string
	var endedAt sql.NullTime
	var updatedAt time.Time
	if err := tx.QueryRowContext(ctx, updateQuery, tournamentID, matchID, payload.ScoreA, payload.ScoreB, payload.WinnerTeamID).Scan(
		&updatedStatus,
		&endedAt,
		&updatedAt,
	); err != nil {
		return matchResultResponse{}, err
	}

	if err := advanceWinnerInTx(ctx, tx, tournamentIDParsed, matchRound, matchIndex, payload.WinnerTeamID); err != nil {
		return matchResultResponse{}, err
	}
	if err := autoAdvanceByesInTx(ctx, tx, tournamentIDParsed); err != nil {
		return matchResultResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return matchResultResponse{}, err
	}

	response := matchResultResponse{
		TournamentID: tournamentIDParsed,
		MatchID:      matchIDParsed,
		Status:       updatedStatus,
		ScoreA:       payload.ScoreA,
		ScoreB:       payload.ScoreB,
		WinnerTeamID: payload.WinnerTeamID,
		UpdatedAt:    updatedAt.UTC().Format(time.RFC3339),
	}
	if endedAt.Valid {
		value := endedAt.Time.UTC().Format(time.RFC3339)
		response.EndedAt = &value
	}
	return response, nil
}

func forfeitMatch(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	matchID string,
	payload forfeitMatchRequest,
) (matchResultResponse, error) {
	if payload.WinnerTeamID == 0 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нужно указать победившую команду",
		}
	}
	reason := strings.ToUpper(strings.TrimSpace(payload.Reason))
	if reason == "" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нужно указать причину",
		}
	}
	if !isAllowedStatusReason(reason) {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "причина не поддерживается",
		}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return matchResultResponse{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const query = `
SELECT
  t.id,
  t.status,
  m.id,
  m.status,
  m.round,
  m.index_in_round,
  m.bo,
  m.team_a_id,
  m.team_b_id
FROM matches m
JOIN tournaments t ON t.id = m.tournament_id
WHERE m.tournament_id = $1 AND m.id = $2`

	var tournamentIDParsed int64
	var tournamentStatus string
	var matchIDParsed int64
	var matchStatus string
	var matchRound int
	var matchIndex int
	var bestOf int
	var teamAID sql.NullInt64
	var teamBID sql.NullInt64
	if err := tx.QueryRowContext(ctx, query, tournamentID, matchID).Scan(
		&tournamentIDParsed,
		&tournamentStatus,
		&matchIDParsed,
		&matchStatus,
		&matchRound,
		&matchIndex,
		&bestOf,
		&teamAID,
		&teamBID,
	); err != nil {
		return matchResultResponse{}, err
	}

	if tournamentStatus != "RUNNING" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "турнир должен быть в статусе Идёт (RUNNING) для технических поражений",
		}
	}
	if matchStatus == "FINISHED" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "матч уже завершён",
		}
	}
	if matchStatus == "CANCELED" {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя обновлять отменённый матч",
		}
	}
	if bestOf < 1 || bestOf%2 == 0 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "некорректный формат серии (BO) матча",
		}
	}
	if !teamAID.Valid || !teamBID.Valid {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "в матче должны быть назначены обе команды",
		}
	}
	if payload.WinnerTeamID != teamAID.Int64 && payload.WinnerTeamID != teamBID.Int64 {
		return matchResultResponse{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "победившая команда должна быть одной из команд матча",
		}
	}

	requiredWins := bestOf/2 + 1
	scoreA := 0
	scoreB := 0
	if payload.WinnerTeamID == teamAID.Int64 {
		scoreA = requiredWins
	} else {
		scoreB = requiredWins
	}

	const updateQuery = `
UPDATE matches
SET
  score_a = $3,
  score_b = $4,
  winner_team_id = $5,
  status = 'FINISHED',
  ended_at = NOW(),
  updated_at = NOW()
WHERE tournament_id = $1 AND id = $2
RETURNING status, ended_at, updated_at`

	var updatedStatus string
	var endedAt sql.NullTime
	var updatedAt time.Time
	if err := tx.QueryRowContext(ctx, updateQuery, tournamentID, matchID, scoreA, scoreB, payload.WinnerTeamID).Scan(
		&updatedStatus,
		&endedAt,
		&updatedAt,
	); err != nil {
		return matchResultResponse{}, err
	}

	if err := advanceWinnerInTx(ctx, tx, tournamentIDParsed, matchRound, matchIndex, payload.WinnerTeamID); err != nil {
		return matchResultResponse{}, err
	}
	if err := autoAdvanceByesInTx(ctx, tx, tournamentIDParsed); err != nil {
		return matchResultResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return matchResultResponse{}, err
	}

	response := matchResultResponse{
		TournamentID: tournamentIDParsed,
		MatchID:      matchIDParsed,
		Status:       updatedStatus,
		ScoreA:       scoreA,
		ScoreB:       scoreB,
		WinnerTeamID: payload.WinnerTeamID,
		UpdatedAt:    updatedAt.UTC().Format(time.RFC3339),
	}
	if endedAt.Valid {
		value := endedAt.Time.UTC().Format(time.RFC3339)
		response.EndedAt = &value
	}
	return response, nil
}

func advanceWinnerInTx(
	ctx context.Context,
	tx *sql.Tx,
	tournamentID int64,
	matchRound int,
	matchIndex int,
	winnerTeamID int64,
) error {
	nextRound := matchRound + 1
	nextIndex := (matchIndex + 1) / 2

	const nextQuery = `
SELECT id, team_a_id, team_b_id
FROM matches
WHERE tournament_id = $1 AND round = $2 AND index_in_round = $3`

	var nextMatchID int64
	var teamAID sql.NullInt64
	var teamBID sql.NullInt64
	if err := tx.QueryRowContext(ctx, nextQuery, tournamentID, nextRound, nextIndex).Scan(
		&nextMatchID,
		&teamAID,
		&teamBID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	assignSide := "A"
	if matchIndex%2 == 0 {
		assignSide = "B"
	}

	if assignSide == "A" {
		if teamAID.Valid && teamAID.Int64 != winnerTeamID {
			return clientError{Status: http.StatusConflict, Message: "слот команды A в следующем матче уже занят"}
		}
		if teamAID.Valid {
			return nil
		}
		_, err := tx.ExecContext(ctx, `UPDATE matches SET team_a_id = $3, updated_at = NOW() WHERE id = $1 AND tournament_id = $2`, nextMatchID, tournamentID, winnerTeamID)
		return err
	}

	if teamBID.Valid && teamBID.Int64 != winnerTeamID {
		return clientError{Status: http.StatusConflict, Message: "слот команды B в следующем матче уже занят"}
	}
	if teamBID.Valid {
		return nil
	}
	_, err := tx.ExecContext(ctx, `UPDATE matches SET team_b_id = $3, updated_at = NOW() WHERE id = $1 AND tournament_id = $2`, nextMatchID, tournamentID, winnerTeamID)
	return err
}

func autoAdvanceByesInTx(ctx context.Context, tx *sql.Tx, tournamentID int64) error {
	for {
		const query = `
SELECT id, round, index_in_round, bo, team_a_id, team_b_id
FROM matches
WHERE tournament_id = $1
  AND status = 'SCHEDULED'
  AND winner_team_id IS NULL
  AND ((team_a_id IS NULL AND team_b_id IS NOT NULL) OR (team_a_id IS NOT NULL AND team_b_id IS NULL))
ORDER BY round ASC, index_in_round ASC`

		rows, err := tx.QueryContext(ctx, query, tournamentID)
		if err != nil {
			return err
		}

		type byeMatch struct {
			ID      int64
			Round   int
			Index   int
			Bo      int
			TeamAID sql.NullInt64
			TeamBID sql.NullInt64
		}
		items := make([]byeMatch, 0)
		for rows.Next() {
			var item byeMatch
			if err := rows.Scan(&item.ID, &item.Round, &item.Index, &item.Bo, &item.TeamAID, &item.TeamBID); err != nil {
				rows.Close()
				return err
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		if len(items) == 0 {
			return nil
		}

		advanced := false
		for _, item := range items {
			missingA := !item.TeamAID.Valid && item.TeamBID.Valid
			missingB := item.TeamAID.Valid && !item.TeamBID.Valid
			if !(missingA || missingB) {
				continue
			}
			if item.Round > 1 {
				if missingA {
					var exists bool
					if err := tx.QueryRowContext(
						ctx,
						`SELECT EXISTS(SELECT 1 FROM matches WHERE tournament_id = $1 AND round = $2 AND index_in_round = $3)`,
						tournamentID,
						item.Round-1,
						item.Index*2-1,
					).Scan(&exists); err != nil {
						return err
					}
					if exists {
						continue
					}
				}
				if missingB {
					var exists bool
					if err := tx.QueryRowContext(
						ctx,
						`SELECT EXISTS(SELECT 1 FROM matches WHERE tournament_id = $1 AND round = $2 AND index_in_round = $3)`,
						tournamentID,
						item.Round-1,
						item.Index*2,
					).Scan(&exists); err != nil {
						return err
					}
					if exists {
						continue
					}
				}
			}

			requiredWins := item.Bo/2 + 1
			var winnerID int64
			scoreA := 0
			scoreB := 0
			if item.TeamAID.Valid {
				winnerID = item.TeamAID.Int64
				scoreA = requiredWins
			} else if item.TeamBID.Valid {
				winnerID = item.TeamBID.Int64
				scoreB = requiredWins
			} else {
				continue
			}

			const updateQuery = `
UPDATE matches
SET
  score_a = $3,
  score_b = $4,
  winner_team_id = $5,
  status = 'FINISHED',
  ended_at = NOW(),
  updated_at = NOW()
WHERE id = $1 AND tournament_id = $2`

			if _, err := tx.ExecContext(ctx, updateQuery, item.ID, tournamentID, scoreA, scoreB, winnerID); err != nil {
				return err
			}
			advanced = true
			if err := advanceWinnerInTx(ctx, tx, tournamentID, item.Round, item.Index, winnerID); err != nil {
				return err
			}
		}
		if !advanced {
			return nil
		}
	}
}

func isValidSidesForGame(game string, first string, second string) bool {
	a, b := defaultSidesForGame(game)
	first = strings.TrimSpace(first)
	second = strings.TrimSpace(second)
	return (first == a && second == b) || (first == b && second == a)
}

func startTournament(ctx context.Context, conn *sql.DB, tournamentID string) (tournament, error) {
	current, err := getTournament(ctx, conn, tournamentID)
	if err != nil {
		return tournament{}, err
	}
	if current.Status == "RUNNING" {
		return tournament{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "турнир уже запущен",
		}
	}
	if current.Status == "FINISHED" {
		return tournament{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя запустить завершённый турнир",
		}
	}

	const matchesCountQuery = `
SELECT COUNT(*)
FROM matches
WHERE tournament_id = $1`
	var matchesCount int
	if err := conn.QueryRowContext(ctx, matchesCountQuery, tournamentID).Scan(&matchesCount); err != nil {
		return tournament{}, err
	}
	if matchesCount == 0 {
		return tournament{}, clientError{
			Status:  http.StatusBadRequest,
			Message: "нельзя запустить турнир без сгенерированной сетки",
		}
	}

	const updateQuery = `
UPDATE tournaments
SET
  status = 'RUNNING',
  is_listed = TRUE,
  is_bracket_published = FALSE,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at`

	var item tournament
	var description sql.NullString
	var startDate sql.NullTime
	var endDate sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time
	err = conn.QueryRowContext(ctx, updateQuery, tournamentID).Scan(
		&item.ID,
		&item.Game,
		&item.Name,
		&description,
		&startDate,
		&endDate,
		&item.AllowOdd,
		&item.Status,
		&item.IsListed,
		&item.IsBracketPublished,
		&item.ScheduleVisibilityAhead,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return tournament{}, err
	}
	if description.Valid {
		item.Description = &description.String
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		item.StartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		item.EndDate = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return item, nil
}

func updateTournamentVisibility(
	ctx context.Context,
	conn *sql.DB,
	tournamentID string,
	payload updateVisibilityRequest,
) (tournament, error) {
	normalizedAhead, err := normalizeScheduleVisibilityAhead(payload.ScheduleVisibility)
	if err != nil {
		return tournament{}, err
	}

	const query = `
UPDATE tournaments
SET
  is_bracket_published = $2,
  schedule_visibility_ahead = $3,
  updated_at = NOW()
WHERE id = $1
RETURNING
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  allow_odd,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at`

	var item tournament
	var description sql.NullString
	var startDate sql.NullTime
	var endDate sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time
	err = conn.QueryRowContext(ctx, query, tournamentID, payload.IsBracketPublished, normalizedAhead).Scan(
		&item.ID,
		&item.Game,
		&item.Name,
		&description,
		&startDate,
		&endDate,
		&item.AllowOdd,
		&item.Status,
		&item.IsListed,
		&item.IsBracketPublished,
		&item.ScheduleVisibilityAhead,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return tournament{}, err
	}
	if description.Valid {
		item.Description = &description.String
	}
	if startDate.Valid {
		value := startDate.Time.Format("2006-01-02")
		item.StartDate = &value
	}
	if endDate.Valid {
		value := endDate.Time.Format("2006-01-02")
		item.EndDate = &value
	}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return item, nil
}

func normalizeScheduleVisibilityAhead(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(value))
	if trimmed == "" {
		return "0", nil
	}
	if trimmed == "ALL" {
		return "ALL", nil
	}
	number, err := strconv.Atoi(trimmed)
	if err != nil || number < 0 {
		return "", clientError{
			Status:  http.StatusBadRequest,
			Message: "параметр видимости расписания должен быть 0, неотрицательным числом или ALL (всё)",
		}
	}
	return strconv.Itoa(number), nil
}

func validateTeamsForTournament(ctx context.Context, conn *sql.DB, tournamentID string) (teamValidationResponse, error) {
	const query = `
SELECT id, allow_odd
FROM tournaments
WHERE id = $1`

	var id int64
	var allowOdd bool
	if err := conn.QueryRowContext(ctx, query, tournamentID).Scan(&id, &allowOdd); err != nil {
		return teamValidationResponse{}, err
	}

	teamCount, err := countTeamsByTournament(ctx, conn, tournamentID)
	if err != nil {
		return teamValidationResponse{}, err
	}

	check := buildTeamConstraintCheck(allowOdd, teamCount)
	return teamValidationResponse{
		TournamentID:    id,
		AllowOdd:        allowOdd,
		TeamCount:       teamCount,
		IsValid:         check.IsValid,
		Message:         check.Message,
		SuggestedAction: check.SuggestedAction,
		BracketSize:     check.BracketSize,
	}, nil
}

func countTeamsByTournament(ctx context.Context, conn *sql.DB, tournamentID string) (int, error) {
	const query = `
SELECT COUNT(*)
FROM teams
WHERE tournament_id = $1`
	var count int
	if err := conn.QueryRowContext(ctx, query, tournamentID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type teamConstraintCheck struct {
	IsValid         bool
	Message         string
	SuggestedAction *string
	BracketSize     *int
}

func (c teamConstraintCheck) MessageWithSuggestion() string {
	if c.SuggestedAction == nil {
		return c.Message
	}
	return c.Message + " " + *c.SuggestedAction
}

func buildTeamConstraintCheck(allowOdd bool, teamCount int) teamConstraintCheck {
	if teamCount < 2 {
		needed := 2 - teamCount
		suggestion := "Добавьте минимум ещё " + strconv.Itoa(needed) + " команд(у)."
		return teamConstraintCheck{
			IsValid:         false,
			Message:         "Нужно минимум 2 команды.",
			SuggestedAction: &suggestion,
		}
	}

	bracketSize := nextPowerOfTwo(teamCount)
	if allowOdd {
		message := "Количество команд подходит. Сетка может включать слоты BYE."
		suggestion := "Ожидаемый размер сетки: " + strconv.Itoa(bracketSize) + "."
		return teamConstraintCheck{
			IsValid:         true,
			Message:         message,
			SuggestedAction: &suggestion,
			BracketSize:     &bracketSize,
		}
	}

	if isPowerOfTwo(teamCount) {
		message := "Количество команд подходит для allowOdd=false."
		return teamConstraintCheck{
			IsValid:     true,
			Message:     message,
			BracketSize: &bracketSize,
		}
	}

	prevPow := previousPowerOfTwo(teamCount)
	addTeams := bracketSize - teamCount
	removeTeams := teamCount - prevPow
	suggestion := "Установите allowOdd=true или измените количество команд до " + strconv.Itoa(prevPow) + " (-" + strconv.Itoa(removeTeams) + ") или " + strconv.Itoa(bracketSize) + " (+" + strconv.Itoa(addTeams) + ")."
	return teamConstraintCheck{
		IsValid:         false,
		Message:         "Для allowOdd=false количество команд должно быть степенью двойки.",
		SuggestedAction: &suggestion,
		BracketSize:     &bracketSize,
	}
}

func isPowerOfTwo(value int) bool {
	return value > 0 && (value&(value-1)) == 0
}

func nextPowerOfTwo(value int) int {
	if value <= 1 {
		return 1
	}
	p := 1
	for p < value {
		p <<= 1
	}
	return p
}

func previousPowerOfTwo(value int) int {
	if value <= 1 {
		return 1
	}
	p := 1
	for (p << 1) <= value {
		p <<= 1
	}
	return p
}

func buildAdminToken(user userRecord) (string, error) {
	secret := getenvDefault("JWT_SECRET", "dev-secret")
	claims := adminClaims{
		AdminID: user.ID,
		Login:   user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func authenticateRequest(r *http.Request) (adminInfo, error) {
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return adminInfo{}, err
	}
	secret := getenvDefault("JWT_SECRET", "dev-secret")
	token, err := jwt.ParseWithClaims(cookie.Value, &adminClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return adminInfo{}, err
	}
	claims, ok := token.Claims.(*adminClaims)
	if !ok || !token.Valid {
		return adminInfo{}, jwt.ErrTokenInvalidClaims
	}
	return adminInfo{ID: claims.AdminID, Login: claims.Login}, nil
}

func setSessionCookie(w http.ResponseWriter, token string) {
	secure := strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
	sameSite := parseSameSite(getenvDefault("COOKIE_SAMESITE", "Lax"))
	cookie := &http.Cookie{
		Name:     "admin_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   60 * 60 * 24,
	}
	if domain := strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(w, cookie)
}

func clearSessionCookie(w http.ResponseWriter) {
	sameSite := parseSameSite(getenvDefault("COOKIE_SAMESITE", "Lax"))
	cookie := &http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: sameSite,
	}
	if domain := strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")); domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(w, cookie)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowed := resolveAllowedOrigins()
		if origin != "" && originAllowed(origin, allowed) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func resolveAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{getenvDefault("FRONTEND_URL", "http://localhost:3000")}
	}
	parts := strings.Split(raw, ",")
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			allowed = append(allowed, value)
		}
	}
	return allowed
}

func originAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		if item == origin {
			return true
		}
	}
	return false
}

func parseSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		value := strings.TrimSpace(parts[0])
		if value != "" {
			return value
		}
	}
	ip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}
