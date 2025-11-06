package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pickem/backend/internal/config"
)

const totalRegularSeasonWeeks = 18

// Run ensures core reference data exists so the API can serve the UI immediately.
func Run(ctx context.Context, pool *pgxpool.Pool, cfg config.Config) error {
	if _, err := ensureFamilyMembers(ctx, pool, cfg.FamilyMembers, cfg.CommissionerName); err != nil {
		return err
	}

	seasonID, err := ensureSeason(ctx, pool, cfg.DefaultSeasonKey)
	if err != nil {
		return err
	}

	if err := ensureSeasonWeeks(ctx, pool, seasonID); err != nil {
		return err
	}

	return nil
}

func ensureFamilyMembers(ctx context.Context, pool *pgxpool.Pool, members []string, commissioner string) (int64, error) {
	if len(members) == 0 {
		return 0, fmt.Errorf("bootstrap: at least one family member must be configured")
	}

	var added int64
	for _, raw := range members {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}

		isCommissioner := strings.EqualFold(name, commissioner)
		res, err := pool.Exec(ctx, `
			insert into family_members (name, is_commissioner)
			values ($1, $2)
			on conflict (name)
			do update set is_commissioner = excluded.is_commissioner
		`, name, isCommissioner)
		if err != nil {
			return added, fmt.Errorf("bootstrap: upsert family member %q: %w", name, err)
		}
		added += res.RowsAffected()
	}

	return added, nil
}

func ensureSeason(ctx context.Context, pool *pgxpool.Pool, seasonKey string) (string, error) {
	seasonKey = strings.TrimSpace(seasonKey)
	if seasonKey == "" {
		return "", fmt.Errorf("bootstrap: SPORTS_SEASON_KEY must be provided")
	}

	year, label, err := seasonLabel(seasonKey)
	if err != nil {
		return "", err
	}

	var seasonID string
	err = pool.QueryRow(ctx, `
		select id
		from seasons
		where sportsdata_season_key = $1
	`, seasonKey).Scan(&seasonID)
	if err == nil {
		return seasonID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("bootstrap: lookup season %s: %w", seasonKey, err)
	}

	err = pool.QueryRow(ctx, `
		insert into seasons (label, season_year, sportsdata_season_key)
		values ($1, $2, $3)
		returning id
	`, label, year, seasonKey).Scan(&seasonID)
	if err != nil {
		return "", fmt.Errorf("bootstrap: insert season %s: %w", seasonKey, err)
	}

	return seasonID, nil
}

func ensureSeasonWeeks(ctx context.Context, pool *pgxpool.Pool, seasonID string) error {
	for week := 1; week <= totalRegularSeasonWeeks; week++ {
		label := fmt.Sprintf("Week %d", week)
		if _, err := pool.Exec(ctx, `
			insert into season_weeks (season_id, number, label)
			values ($1, $2, $3)
			on conflict (season_id, number)
			do update set label = excluded.label
		`, seasonID, week, label); err != nil {
			return fmt.Errorf("bootstrap: ensure week %d: %w", week, err)
		}
	}
	return nil
}

func seasonLabel(seasonKey string) (int, string, error) {
	yearDigits := 0
	for _, r := range seasonKey {
		if !unicode.IsDigit(r) {
			break
		}
		yearDigits++
	}
	if yearDigits == 0 {
		return 0, "", fmt.Errorf("bootstrap: season key %q does not start with a year", seasonKey)
	}

	year, err := strconv.Atoi(seasonKey[:yearDigits])
	if err != nil {
		return 0, "", fmt.Errorf("bootstrap: invalid season year in key %q: %w", seasonKey, err)
	}

	suffix := strings.ToUpper(strings.TrimSpace(seasonKey[yearDigits:]))
	var label string
	switch suffix {
	case "", "REG":
		label = fmt.Sprintf("%d Regular Season", year)
	case "POST":
		label = fmt.Sprintf("%d Postseason", year)
	case "PRE":
		label = fmt.Sprintf("%d Preseason", year)
	default:
		label = fmt.Sprintf("%d %s", year, suffix)
	}

	return year, label, nil
}
