package nfl

import "strings"

// Team represents a simplified NFL team entry.
type Team struct {
	Code     string
	Name     string
	Location string
}

var teamsByCode = map[string]Team{
	"ARI": {Code: "ARI", Name: "Cardinals", Location: "Arizona"},
	"ATL": {Code: "ATL", Name: "Falcons", Location: "Atlanta"},
	"BAL": {Code: "BAL", Name: "Ravens", Location: "Baltimore"},
	"BUF": {Code: "BUF", Name: "Bills", Location: "Buffalo"},
	"CAR": {Code: "CAR", Name: "Panthers", Location: "Carolina"},
	"CHI": {Code: "CHI", Name: "Bears", Location: "Chicago"},
	"CIN": {Code: "CIN", Name: "Bengals", Location: "Cincinnati"},
	"CLE": {Code: "CLE", Name: "Browns", Location: "Cleveland"},
	"DAL": {Code: "DAL", Name: "Cowboys", Location: "Dallas"},
	"DEN": {Code: "DEN", Name: "Broncos", Location: "Denver"},
	"DET": {Code: "DET", Name: "Lions", Location: "Detroit"},
	"GB":  {Code: "GB", Name: "Packers", Location: "Green Bay"},
	"HOU": {Code: "HOU", Name: "Texans", Location: "Houston"},
	"IND": {Code: "IND", Name: "Colts", Location: "Indianapolis"},
	"JAX": {Code: "JAX", Name: "Jaguars", Location: "Jacksonville"},
	"KC":  {Code: "KC", Name: "Chiefs", Location: "Kansas City"},
	"LAC": {Code: "LAC", Name: "Chargers", Location: "Los Angeles"},
	"LAR": {Code: "LAR", Name: "Rams", Location: "Los Angeles"},
	"LVR": {Code: "LVR", Name: "Raiders", Location: "Las Vegas"},
	"MIA": {Code: "MIA", Name: "Dolphins", Location: "Miami"},
	"MIN": {Code: "MIN", Name: "Vikings", Location: "Minnesota"},
	"NE":  {Code: "NE", Name: "Patriots", Location: "New England"},
	"NO":  {Code: "NO", Name: "Saints", Location: "New Orleans"},
	"NYG": {Code: "NYG", Name: "Giants", Location: "New York"},
	"NYJ": {Code: "NYJ", Name: "Jets", Location: "New York"},
	"PHI": {Code: "PHI", Name: "Eagles", Location: "Philadelphia"},
	"PIT": {Code: "PIT", Name: "Steelers", Location: "Pittsburgh"},
	"SEA": {Code: "SEA", Name: "Seahawks", Location: "Seattle"},
	"SF":  {Code: "SF", Name: "49ers", Location: "San Francisco"},
	"TB":  {Code: "TB", Name: "Buccaneers", Location: "Tampa Bay"},
	"TEN": {Code: "TEN", Name: "Titans", Location: "Tennessee"},
	"WAS": {Code: "WAS", Name: "Commanders", Location: "Washington"},
}

// Lookup returns a Team by its code (case insensitive). If the team is unknown
// a generic placeholder is returned so callers always receive a value.
func Lookup(code string) Team {
	upper := strings.ToUpper(strings.TrimSpace(code))
	if upper == "" {
		return Team{Code: "", Name: "Unknown", Location: "Unknown"}
	}
	if team, ok := teamsByCode[upper]; ok {
		return team
	}
	return Team{Code: upper, Name: upper, Location: upper}
}
