package models

import "time"

type Season struct {
	ID                  string `json:"id"`
	Label               string `json:"label"`
	Year                int    `json:"year"`
	SportsDataSeasonKey string `json:"sportsDataSeasonKey"`
}

type Week struct {
	ID       string     `json:"id"`
	Number   int        `json:"number"`
	Label    string     `json:"label"`
	StartsAt *time.Time `json:"startsAt,omitempty"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
}

type RecordSummary struct {
	Wins   int `json:"wins"`
	Losses int `json:"losses"`
}

type Member struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	IsCommissioner bool          `json:"isCommissioner"`
	SeasonRecord   RecordSummary `json:"seasonRecord"`
	LastWeekRecord RecordSummary `json:"lastWeekRecord"`
	WeeksWon       int           `json:"weeksWon"`
	TieBreakers    map[int]int   `json:"tieBreakers"`
}

type TeamInfo struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

type GamePick struct {
	MemberID   string `json:"memberId"`
	ChosenSide string `json:"chosenSide"`
	Status     string `json:"status"`
}

type Game struct {
	ID        string     `json:"id"`
	GameKey   string     `json:"gameKey"`
	Kickoff   *time.Time `json:"kickoff,omitempty"`
	Location  string     `json:"location"`
	Status    string     `json:"status"`
	Channel   string     `json:"channel,omitempty"`
	HomeTeam  TeamInfo   `json:"home"`
	AwayTeam  TeamInfo   `json:"away"`
	HomeScore *int       `json:"homeScore,omitempty"`
	AwayScore *int       `json:"awayScore,omitempty"`
	Winner    string     `json:"winner,omitempty"`
	Picks     []GamePick `json:"picks"`
}

type WeekResult struct {
	SeasonWeekID       string     `json:"seasonWeekId"`
	WinnerMemberID     string     `json:"winnerMemberId,omitempty"`
	DeclaredByMemberID string     `json:"declaredByMemberId,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	DeclaredAt         *time.Time `json:"declaredAt,omitempty"`
}

type PageData struct {
	Season     Season      `json:"season"`
	Weeks      []Week      `json:"weeks"`
	ActiveWeek Week        `json:"activeWeek"`
	Members    []Member    `json:"members"`
	Games      []Game      `json:"games"`
	WeekResult *WeekResult `json:"weekResult,omitempty"`
}
