package scheduler

import (
	"context"
	"log"
	"strings"
	"time"

	"pickem/backend/internal/config"
	"pickem/backend/internal/store"
	"pickem/backend/sportsdata"
)

// CurrentWeekJob periodically syncs the site's default week with SportsData.io.
type CurrentWeekJob struct {
	cfg    config.Config
	store  *store.Store
	cancel context.CancelFunc
}

func NewCurrentWeekJob(cfg config.Config, st *store.Store) *CurrentWeekJob {
	return &CurrentWeekJob{cfg: cfg, store: st}
}

// Start begins the weekly sync loop. It is a no-op if the API key or season key are missing.
func (j *CurrentWeekJob) Start(ctx context.Context) {
	if strings.TrimSpace(j.cfg.SportsAPIKey) == "" || strings.TrimSpace(j.cfg.DefaultSeasonKey) == "" {
		return
	}

	loopCtx, cancel := context.WithCancel(ctx)
	j.cancel = cancel

	go j.loop(loopCtx)
	go j.run(loopCtx)
}

// Stop halts the scheduled loop.
func (j *CurrentWeekJob) Stop() {
	if j.cancel != nil {
		j.cancel()
	}
}

func (j *CurrentWeekJob) loop(ctx context.Context) {
	for {
		nextRun := nextWednesdayAtTwo(time.Now())
		delay := time.Until(nextRun)
		if delay <= 0 {
			delay = time.Hour
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			j.run(ctx)
		}
	}
}

func (j *CurrentWeekJob) run(ctx context.Context) {
	if j.store == nil {
		return
	}

	jobCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	season, err := j.store.GetSeasonBySportsKey(jobCtx, j.cfg.DefaultSeasonKey)
	if err != nil {
		log.Printf("scheduler: current week: lookup season: %v", err)
		return
	}

	week, err := sportsdata.FetchCurrentWeek(jobCtx, nil, j.cfg.SportsAPIBaseURL, j.cfg.SportsAPIKey)
	if err != nil {
		log.Printf("scheduler: current week: fetch: %v", err)
		return
	}

	if err := j.store.SetSeasonCurrentWeek(jobCtx, season.ID, week); err != nil {
		log.Printf("scheduler: current week: set week %d: %v", week, err)
		return
	}

	log.Printf("scheduler: current week updated to Week %d", week)
}

func nextWednesdayAtTwo(now time.Time) time.Time {
	loc := now.Location()
	// Determine days to add to reach Wednesday.
	daysAhead := (int(time.Wednesday) - int(now.Weekday()) + 7) % 7
	targetDate := now.AddDate(0, 0, daysAhead)
	target := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 14, 0, 0, 0, loc)
	if !target.After(now) {
		target = target.AddDate(0, 0, 7)
	}
	return target
}
