package sportsdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const defaultBaseURL = "https://api.sportsdata.io/v3/nfl"

// GameSnapshot is the minimal representation the app needs to render picks and results.
type GameSnapshot struct {
	GameKey   string  `json:"gameKey"`
	Season    int     `json:"season"`
	Week      int     `json:"week"`
	Kickoff   *string `json:"kickoff,omitempty"`
	Channel   string  `json:"channel,omitempty"`
	Location  string  `json:"location,omitempty"`
	HomeTeam  string  `json:"homeTeam"`
	AwayTeam  string  `json:"awayTeam"`
	HomeScore *int    `json:"homeScore,omitempty"`
	AwayScore *int    `json:"awayScore,omitempty"`
	Status    string  `json:"status"`
}

// FetchScoresByWeek retrieves minimal game data for the given season/week.
func FetchScoresByWeek(ctx context.Context, httpClient *http.Client, baseURL, apiKey, seasonKey string, week int) ([]GameSnapshot, error) {
	if apiKey == "" {
		return nil, errors.New("sportsdata: api key must be provided")
	}
	if seasonKey == "" {
		return nil, errors.New("sportsdata: season key must be provided")
	}
	if week <= 0 {
		return nil, errors.New("sportsdata: week must be positive")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	endpoint, err := scoresByWeekURL(baseURL, apiKey, seasonKey, week)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("sportsdata: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sportsdata: request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("sportsdata: unexpected status %d: %s", res.StatusCode, string(body))
	}

	var payload []scoreResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("sportsdata: decode response: %w", err)
	}

	snapshots := make([]GameSnapshot, 0, len(payload))
	for _, score := range payload {
		snapshots = append(snapshots, score.toSnapshot())
	}

	return snapshots, nil
}

func scoresByWeekURL(baseURL, apiKey, seasonKey string, week int) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("sportsdata: invalid base url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/scores/json/ScoresByWeek/%s/%d", u.Path, seasonKey, week)
	query := u.Query()
	query.Set("key", apiKey)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

type scoreResponse struct {
	GameKey     string  `json:"GameKey"`
	Season      int     `json:"Season"`
	Week        int     `json:"Week"`
	DateTime    *string `json:"DateTime"`
	DateTimeUTC *string `json:"DateTimeUTC"`
	Date        *string `json:"Date"`
	Channel     *string `json:"Channel"`
	HomeTeam    string  `json:"HomeTeam"`
	AwayTeam    string  `json:"AwayTeam"`
	HomeScore   *int    `json:"HomeScore"`
	AwayScore   *int    `json:"AwayScore"`
	Status      string  `json:"Status"`
	Stadium     *stadiumDetails `json:"StadiumDetails"`
}

type stadiumDetails struct {
	Name    *string `json:"Name"`
	City    *string `json:"City"`
	State   *string `json:"State"`
	Country *string `json:"Country"`
}

func (s scoreResponse) toSnapshot() GameSnapshot {
	kickoff := firstNonNilString(s.DateTimeUTC, s.DateTime, s.Date)

	location := ""
	if s.Stadium != nil {
		switch {
		case s.Stadium.City != nil && s.Stadium.Name != nil:
			location = fmt.Sprintf("%s (%s)", safeString(s.Stadium.Name), safeString(s.Stadium.City))
		case s.Stadium.Name != nil:
			location = safeString(s.Stadium.Name)
		case s.Stadium.City != nil:
			location = safeString(s.Stadium.City)
		}
	}

	return GameSnapshot{
		GameKey:   s.GameKey,
		Season:    s.Season,
		Week:      s.Week,
		Kickoff:   kickoff,
		Channel:   safeString(s.Channel),
		Location:  location,
		HomeTeam:  s.HomeTeam,
		AwayTeam:  s.AwayTeam,
		HomeScore: s.HomeScore,
		AwayScore: s.AwayScore,
		Status:    s.Status,
	}
}

func firstNonNilString(values ...*string) *string {
	for _, v := range values {
		if v != nil && *v != "" {
			return v
		}
	}
	return nil
}

// WeekFromString converts week path parameters (which arrive as strings in Go HTTP handlers) into integers.
func WeekFromString(raw string) (int, error) {
	week, err := strconv.Atoi(raw)
	if err != nil || week <= 0 {
		return 0, fmt.Errorf("sportsdata: invalid week %q", raw)
	}
	return week, nil
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
