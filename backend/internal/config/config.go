package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration for the Go API service.
type Config struct {
	Port             string
	DatabaseURL      string
	SportsAPIKey     string
	SportsAPIBaseURL string
	DefaultSeasonKey string
	AllowCORSOrigins []string
	EnableSportsSync bool
	FamilyMembers    []string
	CommissionerName string
}

// Load reads configuration from environment variables.
// It is expected that .env has already been processed by the caller (e.g. via godotenv).
func Load() (Config, error) {
	familyMembers := splitAndTrim(os.Getenv("FAMILY_MEMBER_NAMES"), ",")
	if len(familyMembers) == 0 {
		return Config{}, fmt.Errorf("config: FAMILY_MEMBER_NAMES is required")
	}

	commissioner := strings.TrimSpace(os.Getenv("COMMISSIONER_NAME"))
	if commissioner == "" {
		commissioner = familyMembers[0]
	}

	cfg := Config{
		Port:             getEnvOrDefault("PORT", "8080"),
		DatabaseURL:      os.Getenv("SUPABASE_DB_URL"),
		SportsAPIKey:     os.Getenv("SPORTS_API_KEY"),
		SportsAPIBaseURL: getEnvOrDefault("SPORTS_API_BASE_URL", ""),
		DefaultSeasonKey: os.Getenv("SPORTS_SEASON_KEY"),
		FamilyMembers:    familyMembers,
		CommissionerName: commissioner,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: SUPABASE_DB_URL is required")
	}

	if cfg.DefaultSeasonKey == "" {
		return Config{}, fmt.Errorf("config: SPORTS_SEASON_KEY is required")
	}

	if rawOrigins := os.Getenv("API_CORS_ALLOW_ORIGINS"); rawOrigins != "" {
		cfg.AllowCORSOrigins = splitAndTrim(rawOrigins, ",")
	}

	if rawSync := os.Getenv("SPORTS_SYNC_ENABLED"); rawSync != "" {
		value, err := strconv.ParseBool(rawSync)
		if err != nil {
			return Config{}, fmt.Errorf("config: invalid SPORTS_SYNC_ENABLED value: %w", err)
		}
		cfg.EnableSportsSync = value
	} else {
		cfg.EnableSportsSync = cfg.SportsAPIKey != ""
	}

	return cfg, nil
}

func splitAndTrim(value string, sep string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
