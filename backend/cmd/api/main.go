package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"pickem/backend/internal/config"
	"pickem/backend/internal/database"
	httpapi "pickem/backend/internal/http"
	"pickem/backend/internal/store"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	st := store.New(pool)
	defer st.Close()

	srv := httpapi.New(cfg, st)

	addr := ":" + cfg.Port
	log.Printf("Big Dog Pool API listening on %s", addr)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
