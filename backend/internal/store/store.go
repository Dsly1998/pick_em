package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pickem/backend/internal/models"
	"pickem/backend/internal/nfl"
	"pickem/backend/sportsdata"
)

var (
	ErrSeasonNotFound = errors.New("store: season not found")
	ErrWeekNotFound   = errors.New("store: week not found")
	ErrGameNotFound   = errors.New("store: game not found")
	validSides        = map[string]struct{}{
		"home": {},
		"away": {},
	}
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) ListSeasons(ctx context.Context) ([]models.Season, error) {
	rows, err := s.pool.Query(ctx, `
		select id, label, season_year, sportsdata_season_key
		from seasons
		order by season_year desc, created_at desc
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list seasons: %w", err)
	}
	defer rows.Close()

	seasons := []models.Season{}
	for rows.Next() {
		var season models.Season
		if err := rows.Scan(&season.ID, &season.Label, &season.Year, &season.SportsDataSeasonKey); err != nil {
			return nil, fmt.Errorf("store: scan season: %w", err)
		}
		seasons = append(seasons, season)
	}
	return seasons, rows.Err()
}

func (s *Store) ListSeasonWeeks(ctx context.Context, seasonID string) ([]models.Week, error) {
	if _, err := s.getSeason(ctx, seasonID); err != nil {
		return nil, err
	}
	return s.listSeasonWeeks(ctx, seasonID)
}

func (s *Store) GetPageData(ctx context.Context, seasonID string, weekNumber int) (*models.PageData, error) {
	season, err := s.getSeason(ctx, seasonID)
	if err != nil {
		return nil, err
	}

	weeks, err := s.listSeasonWeeks(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	if len(weeks) == 0 {
		return nil, fmt.Errorf("store: season %s has no weeks", season.ID)
	}

	if weekNumber <= 0 {
		weekNumber = weeks[0].Number
	}

	week, err := s.getWeekByNumber(ctx, season.ID, weekNumber)
	if err != nil {
		if errors.Is(err, ErrWeekNotFound) {
			// Fall back to the final week in the list if the requested week is missing.
			week = &weeks[len(weeks)-1]
		} else {
			return nil, err
		}
	}

	page := models.PageData{
		Season:     *season,
		Weeks:      weeks,
		ActiveWeek: *week,
	}

	members, err := s.listMembersWithStats(ctx, season.ID, *week)
	if err != nil {
		return nil, err
	}
	page.Members = members

	games, err := s.listGamesWithPicks(ctx, week.ID)
	if err != nil {
		return nil, err
	}
	page.Games = games

	weekResult, err := s.getWeekResult(ctx, week.ID)
	if err != nil {
		return nil, err
	}
	if weekResult != nil {
		page.WeekResult = weekResult
	}

	return &page, nil
}

func (s *Store) getSeason(ctx context.Context, seasonID string) (*models.Season, error) {
	row := s.pool.QueryRow(ctx, `
		select id, label, season_year, sportsdata_season_key
		from seasons
		where id = $1
	`, seasonID)

	var season models.Season
	if err := row.Scan(&season.ID, &season.Label, &season.Year, &season.SportsDataSeasonKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSeasonNotFound
		}
		return nil, fmt.Errorf("store: get season: %w", err)
	}
	return &season, nil
}

func (s *Store) GetSeason(ctx context.Context, seasonID string) (*models.Season, error) {
	return s.getSeason(ctx, seasonID)
}

func (s *Store) GetSeasonBySportsKey(ctx context.Context, sportsKey string) (*models.Season, error) {
	sportsKey = strings.TrimSpace(sportsKey)
	if sportsKey == "" {
		return nil, fmt.Errorf("store: sports key is required")
	}

	row := s.pool.QueryRow(ctx, `
		select id, label, season_year, sportsdata_season_key
		from seasons
		where sportsdata_season_key = $1
	`, sportsKey)

	var season models.Season
	if err := row.Scan(&season.ID, &season.Label, &season.Year, &season.SportsDataSeasonKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSeasonNotFound
		}
		return nil, fmt.Errorf("store: get season by key: %w", err)
	}
	return &season, nil
}

func (s *Store) listSeasonWeeks(ctx context.Context, seasonID string) ([]models.Week, error) {
	rows, err := s.pool.Query(ctx, `
		select id, number, label, starts_at, ends_at
		from season_weeks
		where season_id = $1
		order by number asc
	`, seasonID)
	if err != nil {
		return nil, fmt.Errorf("store: list weeks: %w", err)
	}
	defer rows.Close()

	var weeks []models.Week
	for rows.Next() {
		var wk models.Week
		if err := rows.Scan(&wk.ID, &wk.Number, &wk.Label, &wk.StartsAt, &wk.EndsAt); err != nil {
			return nil, fmt.Errorf("store: scan week: %w", err)
		}
		weeks = append(weeks, wk)
	}
	return weeks, rows.Err()
}

func (s *Store) GetSeasonCurrentWeek(ctx context.Context, seasonID string) (int, error) {
	if _, err := s.getSeason(ctx, seasonID); err != nil {
		return 0, err
	}

	row := s.pool.QueryRow(ctx, `
		select current_week
		from season_settings
		where season_id = $1
	`, seasonID)

	var week int
	if err := row.Scan(&week); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 1, nil
		}
		return 0, fmt.Errorf("store: get season current week: %w", err)
	}
	if week <= 0 {
		return 1, nil
	}
	return week, nil
}

func (s *Store) SetSeasonCurrentWeek(ctx context.Context, seasonID string, weekNumber int) error {
	if weekNumber <= 0 {
		return fmt.Errorf("store: current week must be positive")
	}

	if _, err := s.getSeason(ctx, seasonID); err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `
		insert into season_settings (season_id, current_week)
		values ($1, $2)
		on conflict (season_id)
		do update set current_week = excluded.current_week, updated_at = now()
	`, seasonID, weekNumber); err != nil {
		return fmt.Errorf("store: set current week: %w", err)
	}
	return nil
}

func (s *Store) getWeekByNumber(ctx context.Context, seasonID string, weekNumber int) (*models.Week, error) {
	row := s.pool.QueryRow(ctx, `
		select id, number, label, starts_at, ends_at
		from season_weeks
		where season_id = $1 and number = $2
	`, seasonID, weekNumber)

	var wk models.Week
	if err := row.Scan(&wk.ID, &wk.Number, &wk.Label, &wk.StartsAt, &wk.EndsAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWeekNotFound
		}
		return nil, fmt.Errorf("store: get week %d for season %s: %w", weekNumber, seasonID, err)
	}
	return &wk, nil
}

func (s *Store) GetWeek(ctx context.Context, seasonID string, weekNumber int) (*models.Week, error) {
	if _, err := s.getSeason(ctx, seasonID); err != nil {
		return nil, err
	}
	return s.getWeekByNumber(ctx, seasonID, weekNumber)
}

func (s *Store) listMembersWithStats(ctx context.Context, seasonID string, activeWeek models.Week) ([]models.Member, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, is_commissioner
		from family_members
		order by name asc
	`)
	if err != nil {
		return nil, fmt.Errorf("store: list members: %w", err)
	}
	defer rows.Close()

	members := []models.Member{}
	memberIndex := map[string]int{}
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.ID, &m.Name, &m.IsCommissioner); err != nil {
			return nil, fmt.Errorf("store: scan member: %w", err)
		}
		m.TieBreakers = map[int]int{}
		memberIndex[m.ID] = len(members)
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return members, nil
	}

	if err := s.populateSeasonRecords(ctx, seasonID, memberIndex, members); err != nil {
		return nil, err
	}

	if err := s.populateLastWeekRecords(ctx, seasonID, activeWeek.Number-1, memberIndex, members); err != nil {
		return nil, err
	}

	if err := s.populateWeeksWon(ctx, seasonID, memberIndex, members); err != nil {
		return nil, err
	}

	if err := s.populateTieBreakers(ctx, seasonID, memberIndex, members); err != nil {
		return nil, err
	}

	return members, nil
}

func (s *Store) populateSeasonRecords(ctx context.Context, seasonID string, memberIndex map[string]int, members []models.Member) error {
	rows, err := s.pool.Query(ctx, `
		select p.member_id,
			sum(case when g.status = 'final' and g.winner = p.chosen_side then 1 else 0 end) as wins,
			sum(case when g.status = 'final' and g.winner is not null and g.winner <> p.chosen_side then 1 else 0 end) as losses
		from picks p
			join games g on g.id = p.game_id
			join season_weeks w on w.id = g.season_week_id
		where w.season_id = $1
		group by p.member_id
	`, seasonID)
	if err != nil {
		return fmt.Errorf("store: season records: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memberID string
		var wins, losses int
		if err := rows.Scan(&memberID, &wins, &losses); err != nil {
			return fmt.Errorf("store: season record scan: %w", err)
		}
		if idx, ok := memberIndex[memberID]; ok {
			members[idx].SeasonRecord = models.RecordSummary{Wins: wins, Losses: losses}
		}
	}
	return rows.Err()
}

func (s *Store) populateLastWeekRecords(ctx context.Context, seasonID string, weekNumber int, memberIndex map[string]int, members []models.Member) error {
	if weekNumber <= 0 {
		return nil
	}

	rows, err := s.pool.Query(ctx, `
		select p.member_id,
			sum(case when g.status = 'final' and g.winner = p.chosen_side then 1 else 0 end) as wins,
			sum(case when g.status = 'final' and g.winner is not null and g.winner <> p.chosen_side then 1 else 0 end) as losses
		from picks p
			join games g on g.id = p.game_id
			join season_weeks w on w.id = g.season_week_id
		where w.season_id = $1
			and w.number = $2
		group by p.member_id
	`, seasonID, weekNumber)
	if err != nil {
		return fmt.Errorf("store: last week records: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memberID string
		var wins, losses int
		if err := rows.Scan(&memberID, &wins, &losses); err != nil {
			return fmt.Errorf("store: last week record scan: %w", err)
		}
		if idx, ok := memberIndex[memberID]; ok {
			members[idx].LastWeekRecord = models.RecordSummary{Wins: wins, Losses: losses}
		}
	}
	return rows.Err()
}

func (s *Store) populateWeeksWon(ctx context.Context, seasonID string, memberIndex map[string]int, members []models.Member) error {
	rows, err := s.pool.Query(ctx, `
		select wr.winner_member_id, count(*)
		from week_results wr
			join season_weeks w on w.id = wr.season_week_id
		where w.season_id = $1
			and wr.winner_member_id is not null
		group by wr.winner_member_id
	`, seasonID)
	if err != nil {
		return fmt.Errorf("store: weeks won: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memberID string
		var wins int
		if err := rows.Scan(&memberID, &wins); err != nil {
			return fmt.Errorf("store: weeks won scan: %w", err)
		}
		if idx, ok := memberIndex[memberID]; ok {
			members[idx].WeeksWon = wins
		}
	}
	return rows.Err()
}

func (s *Store) populateTieBreakers(ctx context.Context, seasonID string, memberIndex map[string]int, members []models.Member) error {
	rows, err := s.pool.Query(ctx, `
		select tb.member_id, w.number, tb.points
		from tie_breakers tb
			join season_weeks w on w.id = tb.season_week_id
		where w.season_id = $1
	`, seasonID)
	if err != nil {
		return fmt.Errorf("store: tie breakers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memberID string
		var weekNumber int
		var points int
		if err := rows.Scan(&memberID, &weekNumber, &points); err != nil {
			return fmt.Errorf("store: tie breaker scan: %w", err)
		}
		if idx, ok := memberIndex[memberID]; ok {
			if members[idx].TieBreakers == nil {
				members[idx].TieBreakers = map[int]int{}
			}
			members[idx].TieBreakers[weekNumber] = points
		}
	}
	return rows.Err()
}

func (s *Store) getWeekResult(ctx context.Context, seasonWeekID string) (*models.WeekResult, error) {
	row := s.pool.QueryRow(ctx, `
		select season_week_id, winner_member_id, declared_by_member_id, notes, declared_at
		from week_results
		where season_week_id = $1
	`, seasonWeekID)

	var result models.WeekResult
	var winnerMemberID *string
	var declaredByMemberID *string
	var notes *string
	if err := row.Scan(&result.SeasonWeekID, &winnerMemberID, &declaredByMemberID, &notes, &result.DeclaredAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get week result: %w", err)
	}
	result.WinnerMemberID = derefString(winnerMemberID)
	result.DeclaredByMemberID = derefString(declaredByMemberID)
	result.Notes = derefString(notes)
	return &result, nil
}

func (s *Store) listGamesWithPicks(ctx context.Context, seasonWeekID string) ([]models.Game, error) {
	rows, err := s.pool.Query(ctx, `
		select
			g.id,
			g.game_key,
			g.kickoff,
			g.status,
			g.channel,
			g.location,
			g.home_team,
			g.away_team,
			g.home_score,
			g.away_score,
			g.winner,
			p.member_id,
			p.chosen_side
		from games g
			left join picks p on p.game_id = g.id
		where g.season_week_id = $1
		order by g.kickoff nulls last, g.game_key asc
	`, seasonWeekID)
	if err != nil {
		return nil, fmt.Errorf("store: list games: %w", err)
	}
	defer rows.Close()

	games := []models.Game{}
	gameIndex := map[string]int{}

	for rows.Next() {
		var (
			gameID     string
			gameKey    string
			kickoff    *time.Time
			status     string
			channel    *string
			location   *string
			homeTeamB  []byte
			awayTeamB  []byte
			homeScore  *int
			awayScore  *int
			winner     *string
			memberID   *string
			chosenSide *string
		)

		if err := rows.Scan(
			&gameID,
			&gameKey,
			&kickoff,
			&status,
			&channel,
			&location,
			&homeTeamB,
			&awayTeamB,
			&homeScore,
			&awayScore,
			&winner,
			&memberID,
			&chosenSide,
		); err != nil {
			return nil, fmt.Errorf("store: scan game row: %w", err)
		}

		idx, exists := gameIndex[gameID]
		if !exists {
			game := models.Game{
				ID:        gameID,
				GameKey:   gameKey,
				Kickoff:   kickoff,
				Status:    status,
				Channel:   derefString(channel),
				Location:  derefString(location),
				HomeScore: homeScore,
				AwayScore: awayScore,
				Winner:    derefString(winner),
			}

			if err := json.Unmarshal(homeTeamB, &game.HomeTeam); err != nil {
				return nil, fmt.Errorf("store: decode home team for game %s: %w", gameKey, err)
			}
			if err := json.Unmarshal(awayTeamB, &game.AwayTeam); err != nil {
				return nil, fmt.Errorf("store: decode away team for game %s: %w", gameKey, err)
			}

			game.Picks = []models.GamePick{}

			games = append(games, game)
			idx = len(games) - 1
			gameIndex[gameID] = idx
		}

		if memberID != nil && chosenSide != nil {
			pick := models.GamePick{
				MemberID:   *memberID,
				ChosenSide: *chosenSide,
				Status:     pickStatus(games[idx], *chosenSide),
			}
			games[idx].Picks = append(games[idx].Picks, pick)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

func pickStatus(game models.Game, side string) string {
	if strings.EqualFold(game.Status, "final") && game.Winner != "" {
		if game.Winner == side {
			return "correct"
		}
		return "incorrect"
	}
	return "pending"
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Store) UpsertPick(ctx context.Context, memberID, gameKey, chosenSide string) (*models.GamePick, error) {
	if _, ok := validSides[chosenSide]; !ok {
		return nil, fmt.Errorf("store: invalid side %q", chosenSide)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("store: upsert pick begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var gameID string
	var gameStatus string
	var winner *string
	err = tx.QueryRow(ctx, `
		select id, status, winner
		from games
		where game_key = $1
	`, gameKey).Scan(&gameID, &gameStatus, &winner)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store: unknown game %s", gameKey)
		}
		return nil, fmt.Errorf("store: query game: %w", err)
	}

	_, err = tx.Exec(ctx, `
		insert into picks (member_id, game_id, chosen_side)
		values ($1, $2, $3)
		on conflict (member_id, game_id)
		do update set chosen_side = excluded.chosen_side, updated_at = now()
	`, memberID, gameID, chosenSide)
	if err != nil {
		return nil, fmt.Errorf("store: upsert pick: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: upsert pick commit: %w", err)
	}

	pick := &models.GamePick{
		MemberID:   memberID,
		ChosenSide: chosenSide,
		Status:     pickStateFromStatus(gameStatus, winner, chosenSide),
	}
	return pick, nil
}

func (s *Store) DeletePick(ctx context.Context, memberID, gameKey string) error {
	if strings.TrimSpace(memberID) == "" || strings.TrimSpace(gameKey) == "" {
		return fmt.Errorf("store: delete pick requires member and game key")
	}

	_, err := s.pool.Exec(ctx, `
		delete from picks p
		using games g
		where p.game_id = g.id
			and p.member_id = $1
			and g.game_key = $2
	`, memberID, gameKey)
	if err != nil {
		return fmt.Errorf("store: delete pick: %w", err)
	}

	return nil
}

func pickStateFromStatus(gameStatus string, winner *string, chosenSide string) string {
	if strings.EqualFold(gameStatus, "final") && winner != nil && *winner != "" {
		if *winner == chosenSide {
			return "correct"
		}
		return "incorrect"
	}
	return "pending"
}

func (s *Store) UpsertTieBreaker(ctx context.Context, memberID, seasonWeekID string, points int) error {
	_, err := s.pool.Exec(ctx, `
		insert into tie_breakers (member_id, season_week_id, points)
		values ($1, $2, $3)
		on conflict (member_id, season_week_id)
		do update set points = excluded.points, updated_at = now()
	`, memberID, seasonWeekID, points)
	if err != nil {
		return fmt.Errorf("store: upsert tie breaker: %w", err)
	}
	return nil
}

func (s *Store) DeclareWeekWinner(ctx context.Context, seasonWeekID, winnerMemberID, declaredByMemberID, notes string) (*models.WeekResult, error) {
	const sql = `
		insert into week_results (season_week_id, winner_member_id, declared_by_member_id, notes)
		values ($1, $2, $3, nullif($4, ''))
		on conflict (season_week_id)
		do update set winner_member_id = excluded.winner_member_id,
			declared_by_member_id = excluded.declared_by_member_id,
			notes = excluded.notes,
			declared_at = now()
		returning season_week_id, winner_member_id, declared_by_member_id, notes, declared_at
	`

	row := s.pool.QueryRow(ctx, sql, seasonWeekID, nullIfEmpty(winnerMemberID), nullIfEmpty(declaredByMemberID), notes)

	var result models.WeekResult
	var outWinner *string
	var outDeclaredBy *string
	var outNotes *string
	if err := row.Scan(&result.SeasonWeekID, &outWinner, &outDeclaredBy, &outNotes, &result.DeclaredAt); err != nil {
		return nil, fmt.Errorf("store: declare week winner: %w", err)
	}
	result.WinnerMemberID = derefString(outWinner)
	result.DeclaredByMemberID = derefString(outDeclaredBy)
	result.Notes = derefString(outNotes)

	return &result, nil
}

func (s *Store) UpdateGameWinner(ctx context.Context, seasonWeekID, gameKey, winner string) (*models.Game, error) {
	gameKey = strings.TrimSpace(gameKey)
	if gameKey == "" {
		return nil, fmt.Errorf("store: update game winner requires game key")
	}

	winner = strings.ToLower(strings.TrimSpace(winner))
	if winner != "" {
		if _, ok := validSides[winner]; !ok {
			return nil, fmt.Errorf("store: invalid winner %q", winner)
		}
	}

	row := s.pool.QueryRow(ctx, `
		update games
		set
			winner = nullif($3, ''),
			status = case when $3 <> '' then 'final' else status end,
			updated_at = now()
		where season_week_id = $1 and game_key = $2
		returning id, game_key, kickoff, status, channel, location, home_team, away_team, home_score, away_score, winner
	`, seasonWeekID, gameKey, winner)

	var (
		game       models.Game
		kickoff    *time.Time
		channel    *string
		location   *string
		homeTeamB  []byte
		awayTeamB  []byte
		homeScore  *int
		awayScore  *int
		winnerText *string
	)

	if err := row.Scan(
		&game.ID,
		&game.GameKey,
		&kickoff,
		&game.Status,
		&channel,
		&location,
		&homeTeamB,
		&awayTeamB,
		&homeScore,
		&awayScore,
		&winnerText,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("store: update game winner: %w", err)
	}

	game.Kickoff = kickoff
	game.Channel = derefString(channel)
	game.Location = derefString(location)
	if err := json.Unmarshal(homeTeamB, &game.HomeTeam); err != nil {
		return nil, fmt.Errorf("store: decode home team for game %s: %w", gameKey, err)
	}
	if err := json.Unmarshal(awayTeamB, &game.AwayTeam); err != nil {
		return nil, fmt.Errorf("store: decode away team for game %s: %w", gameKey, err)
	}
	game.HomeScore = homeScore
	game.AwayScore = awayScore
	game.Winner = derefString(winnerText)
	game.Picks = []models.GamePick{}

	return &game, nil
}

func nullIfEmpty(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (s *Store) SyncWeekFromSnapshots(ctx context.Context, season models.Season, week models.Week, snapshots []sportsdata.GameSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("store: sync week begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, snap := range snapshots {
		homeTeam := nfl.Lookup(snap.HomeTeam)
		awayTeam := nfl.Lookup(snap.AwayTeam)

		homeTeamJSON, err := json.Marshal(homeTeam)
		if err != nil {
			return fmt.Errorf("store: sync week marshal home team: %w", err)
		}
		awayTeamJSON, err := json.Marshal(awayTeam)
		if err != nil {
			return fmt.Errorf("store: sync week marshal away team: %w", err)
		}

		status := normalizeGameStatus(snap.Status)
		winner := ""
		if status == "final" && snap.HomeScore != nil && snap.AwayScore != nil {
			if *snap.HomeScore > *snap.AwayScore {
				winner = "home"
			} else if *snap.AwayScore > *snap.HomeScore {
				winner = "away"
			}
		}

		rawPayload, err := json.Marshal(snap)
		if err != nil {
			return fmt.Errorf("store: sync week marshal payload: %w", err)
		}

		_, err = tx.Exec(ctx, `
			insert into games (
				season_week_id,
				game_key,
				kickoff,
				status,
				channel,
				location,
				home_team,
				away_team,
				home_score,
				away_score,
				winner,
				sportsdata_payload
			)
			values ($1, $2, $3, $4, nullif($5, ''), nullif($6, ''), $7, $8, $9, $10, nullif($11, ''), $12)
			on conflict (game_key)
			do update set
				season_week_id = excluded.season_week_id,
				kickoff = excluded.kickoff,
				status = case when games.status = 'final' then games.status else excluded.status end,
				channel = excluded.channel,
				location = excluded.location,
				home_team = excluded.home_team,
				away_team = excluded.away_team,
				home_score = excluded.home_score,
				away_score = excluded.away_score,
				winner = coalesce(excluded.winner, games.winner),
				sportsdata_payload = excluded.sportsdata_payload,
				updated_at = now()
		`,
			week.ID,
			snap.GameKey,
			parseOptionalTime(snap.Kickoff),
			status,
			snap.Channel,
			snap.Location,
			homeTeamJSON,
			awayTeamJSON,
			snap.HomeScore,
			snap.AwayScore,
			winner,
			rawPayload,
		)
		if err != nil {
			return fmt.Errorf("store: sync week upsert game %s: %w", snap.GameKey, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: sync week commit: %w", err)
	}

	return nil
}

func normalizeGameStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "inprogress", "in-progress", "in progress", "playing":
		return "in-progress"
	case "final", "complete", "completed":
		return "final"
	default:
		return "scheduled"
	}
}

func parseOptionalTime(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	layouts := []struct {
		layout string
		useUTC bool
	}{
		{time.RFC3339, false},
		{"2006-01-02T15:04:05", true},
	}
	for _, candidate := range layouts {
		var parsed time.Time
		var err error
		if candidate.useUTC {
			parsed, err = time.ParseInLocation(candidate.layout, trimmed, time.UTC)
		} else {
			parsed, err = time.Parse(candidate.layout, trimmed)
		}
		if err == nil {
			return &parsed
		}
	}
	return nil
}
