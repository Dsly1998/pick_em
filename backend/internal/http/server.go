package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"pickem/backend/internal/config"
	"pickem/backend/internal/models"
	"pickem/backend/internal/store"
	"pickem/backend/sportsdata"
)

type Server struct {
	cfg    config.Config
	store  *store.Store
	router chi.Router
}

var (
	errSportsAPIKeyMissing   = errors.New("sports api key is not configured")
	errSportsDataUnavailable = errors.New("sports data unavailable")
)

func New(cfg config.Config, store *store.Store) *Server {
	s := &Server{
		cfg:    cfg,
		store:  store,
		router: chi.NewRouter(),
	}

	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.Recoverer)

	if len(cfg.AllowCORSOrigins) > 0 {
		s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.AllowCORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	} else {
		s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	s.routes()

	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.Get("/healthz", s.handleHealth)
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/seasons", s.handleListSeasons)
		r.Get("/seasons/{seasonID}/weeks", s.handleListSeasonWeeks)
		r.Get("/seasons/{seasonID}/weeks/{weekNumber}", s.handleGetPageData)
		r.Post("/seasons/{seasonID}/weeks/{weekNumber}/picks", s.handleUpsertPick)
		r.Delete("/seasons/{seasonID}/weeks/{weekNumber}/picks", s.handleDeletePick)
		r.Post("/seasons/{seasonID}/weeks/{weekNumber}/tie-breaker", s.handleUpsertTieBreaker)
		r.Post("/seasons/{seasonID}/weeks/{weekNumber}/games/{gameKey}/winner", s.handleSetGameWinner)
		r.Post("/seasons/{seasonID}/weeks/{weekNumber}/winner", s.handleDeclareWinner)
		r.Post("/seasons/{seasonID}/weeks/{weekNumber}/sync", s.handleSyncWeek)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListSeasons(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasons, err := s.store.ListSeasons(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"seasons": seasons})
}

func (s *Server) handleListSeasonWeeks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID := chi.URLParam(r, "seasonID")
	if seasonID == "" {
		writeError(w, http.StatusBadRequest, errors.New("seasonID is required"))
		return
	}

	weeks, err := s.store.ListSeasonWeeks(ctx, seasonID)
	if err != nil {
		if errors.Is(err, store.ErrSeasonNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"weeks": weeks})
}

func (s *Server) handleGetPageData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	data, err := s.store.GetPageData(ctx, seasonID, weekNumber)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrSeasonNotFound), errors.Is(err, store.ErrWeekNotFound):
			writeError(w, http.StatusNotFound, err)
			return
		default:
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if len(data.Games) == 0 && s.cfg.EnableSportsSync && s.cfg.SportsAPIKey != "" {
		season := data.Season
		week := data.ActiveWeek
		snapshots, syncErr := s.syncWeek(ctx, &season, &week)
		if syncErr != nil && !errors.Is(syncErr, errSportsAPIKeyMissing) {
			log.Printf("http: automatic sync failed for season %s week %d: %v", seasonID, week.Number, syncErr)
		}
		if syncErr == nil && len(snapshots) > 0 {
			if refreshed, err := s.store.GetPageData(ctx, seasonID, week.Number); err == nil {
				data = refreshed
			} else {
				log.Printf("http: reload after sync failed for season %s week %d: %v", seasonID, week.Number, err)
			}
		}
	}

	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleUpsertPick(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if _, err := s.store.GetWeek(ctx, seasonID, weekNumber); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrSeasonNotFound) || errors.Is(err, store.ErrWeekNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	var req pickRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.MemberID = strings.TrimSpace(req.MemberID)
	req.GameKey = strings.TrimSpace(req.GameKey)
	req.Side = strings.ToLower(strings.TrimSpace(req.Side))

	if req.MemberID == "" || req.GameKey == "" || req.Side == "" {
		writeError(w, http.StatusBadRequest, errors.New("memberId, gameKey, and side are required"))
		return
	}

	pick, err := s.store.UpsertPick(ctx, req.MemberID, req.GameKey, req.Side)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"pick": pick})
}

func (s *Server) handleDeletePick(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if _, err := s.store.GetWeek(ctx, seasonID, weekNumber); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrSeasonNotFound) || errors.Is(err, store.ErrWeekNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	var req deletePickRequest
	if err := decodeJSON(r, &req); err != nil {
		if !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}

	if req.MemberID == "" {
		req.MemberID = r.URL.Query().Get("memberId")
	}
	if req.GameKey == "" {
		req.GameKey = r.URL.Query().Get("gameKey")
	}

	req.MemberID = strings.TrimSpace(req.MemberID)
	req.GameKey = strings.TrimSpace(req.GameKey)

	if req.MemberID == "" || req.GameKey == "" {
		writeError(w, http.StatusBadRequest, errors.New("memberId and gameKey are required"))
		return
	}

	if err := s.store.DeletePick(ctx, req.MemberID, req.GameKey); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"removed": true})
}

func (s *Server) handleUpsertTieBreaker(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	week, err := s.store.GetWeek(ctx, seasonID, weekNumber)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrSeasonNotFound) || errors.Is(err, store.ErrWeekNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	var req tieBreakerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.MemberID = strings.TrimSpace(req.MemberID)
	if req.MemberID == "" {
		writeError(w, http.StatusBadRequest, errors.New("memberId is required"))
		return
	}

	if err := s.store.UpsertTieBreaker(ctx, req.MemberID, week.ID, req.Points); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tieBreaker": map[string]any{
			"memberId":   req.MemberID,
			"weekNumber": week.Number,
			"points":     req.Points,
		},
	})
}

func (s *Server) handleSetGameWinner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	gameKey := chi.URLParam(r, "gameKey")
	if strings.TrimSpace(gameKey) == "" {
		writeError(w, http.StatusBadRequest, errors.New("gameKey is required"))
		return
	}

	week, err := s.store.GetWeek(ctx, seasonID, weekNumber)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, store.ErrSeasonNotFound), errors.Is(err, store.ErrWeekNotFound):
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	var req setGameWinnerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	winner := ""
	if req.Winner != nil {
		winner = *req.Winner
	}

	game, err := s.store.UpdateGameWinner(ctx, week.ID, gameKey, winner)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrGameNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"game": game})
}

func (s *Server) handleDeclareWinner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	week, err := s.store.GetWeek(ctx, seasonID, weekNumber)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrSeasonNotFound) || errors.Is(err, store.ErrWeekNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	var req declareWinnerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.DeclaredByMemberID = strings.TrimSpace(req.DeclaredByMemberID)
	req.WinnerMemberID = strings.TrimSpace(req.WinnerMemberID)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.DeclaredByMemberID == "" {
		writeError(w, http.StatusBadRequest, errors.New("declaredByMemberId is required"))
		return
	}

	result, err := s.store.DeclareWeekWinner(ctx, week.ID, req.WinnerMemberID, req.DeclaredByMemberID, req.Notes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"weekResult": result})
}

func (s *Server) handleSyncWeek(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seasonID, weekNumber, err := parseSeasonWeekParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	season, err := s.store.GetSeason(ctx, seasonID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrSeasonNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	week, err := s.store.GetWeek(ctx, seasonID, weekNumber)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, store.ErrWeekNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	snapshots, err := s.syncWeek(ctx, season, week)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errSportsAPIKeyMissing):
			status = http.StatusInternalServerError
		case errors.Is(err, errSportsDataUnavailable):
			status = http.StatusBadGateway
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"syncedGames": len(snapshots),
	})
}

func (s *Server) syncWeek(ctx context.Context, season *models.Season, week *models.Week) ([]sportsdata.GameSnapshot, error) {
	if season == nil || week == nil {
		return nil, fmt.Errorf("sync: season and week must be provided")
	}

	if strings.TrimSpace(s.cfg.SportsAPIKey) == "" {
		return nil, errSportsAPIKeyMissing
	}

	if strings.TrimSpace(season.SportsDataSeasonKey) == "" {
		return nil, fmt.Errorf("sync: season %s is missing a sports data key", season.ID)
	}

	if week.Number <= 0 {
		return nil, fmt.Errorf("sync: invalid week number %d", week.Number)
	}

	snapshots, err := sportsdata.FetchScoresByWeek(ctx, nil, s.cfg.SportsAPIBaseURL, s.cfg.SportsAPIKey, season.SportsDataSeasonKey, week.Number)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errSportsDataUnavailable, err)
	}

	if err := s.store.SyncWeekFromSnapshots(ctx, *season, *week, snapshots); err != nil {
		return nil, err
	}

	return snapshots, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func parseSeasonWeekParams(r *http.Request) (string, int, error) {
	seasonID := chi.URLParam(r, "seasonID")
	if strings.TrimSpace(seasonID) == "" {
		return "", 0, errors.New("seasonID is required")
	}

	weekStr := chi.URLParam(r, "weekNumber")
	if strings.TrimSpace(weekStr) == "" {
		return "", 0, errors.New("weekNumber is required")
	}

	weekNumber, err := strconv.Atoi(weekStr)
	if err != nil || weekNumber <= 0 {
		return "", 0, errors.New("weekNumber must be a positive integer")
	}

	return seasonID, weekNumber, nil
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

type pickRequest struct {
	MemberID string `json:"memberId"`
	GameKey  string `json:"gameKey"`
	Side     string `json:"side"`
}

type tieBreakerRequest struct {
	MemberID string `json:"memberId"`
	Points   int    `json:"points"`
}

type setGameWinnerRequest struct {
	Winner *string `json:"winner"`
}

type declareWinnerRequest struct {
	WinnerMemberID     string `json:"winnerMemberId"`
	DeclaredByMemberID string `json:"declaredByMemberId"`
	Notes              string `json:"notes"`
}

type deletePickRequest struct {
	MemberID string `json:"memberId"`
	GameKey  string `json:"gameKey"`
}
