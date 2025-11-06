# Big Dog Pool

Pick'em tracker for one family. The Go API reads NFL scores from SportsData.io, stores everything in Supabase Postgres, and the SvelteKit UI renders/edits communal picks.

## Prerequisites

- Go 1.22+
- Node 20+
- A Supabase project (or any Postgres-compatible URL)
- SportsData.io NFL API key

## Environment

Set the variables in both `.env` (frontend) and `backend/.env` (API). The important ones are:

```env
PUBLIC_API_BASE_URL=http://localhost:8080
PUBLIC_SUPABASE_URL=...
PUBLIC_SUPABASE_ANON_KEY=...

SUPABASE_DB_URL=postgresql://...
SPORTS_API_KEY=...
SPORTS_API_BASE_URL=https://api.sportsdata.io/v3/nfl
SPORTS_SEASON_KEY=2025REG
FAMILY_MEMBER_NAMES=Dallin,Danielle,Lauren,Brad,Dad,Mom
COMMISSIONER_NAME=Brad
SPORTS_SYNC_ENABLED=true
```

On startup the API uses those values to:

- Upsert the family roster (flagging the commissioner)
- Ensure the season and 18 regular-season weeks exist
- Auto-sync games from SportsData when a week is opened (and on-demand via the “Sync Week” button)

## Install & Run

```sh
# Backend
cd backend
go run ./cmd/api

# Frontend (second terminal)
cd ..
npm install
npm run dev -- --open
```

The UI talks to the Go API at `http://localhost:8080`. Picks, tie breakers, and winners are shared for the whole family—no auth required.
