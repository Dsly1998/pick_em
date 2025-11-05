-- Enable extensions required for UUID generation (Supabase has them by default, but this keeps the SQL self-contained)
create extension if not exists "uuid-ossp";
create extension if not exists "pgcrypto";

create table if not exists family_members (
	id uuid primary key default gen_random_uuid(),
	name text not null unique,
	is_commissioner boolean not null default false,
	created_at timestamptz not null default now()
);

create table if not exists seasons (
	id uuid primary key default gen_random_uuid(),
	label text not null,
	season_year int not null,
	sportsdata_season_key text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists season_weeks (
	id uuid primary key default gen_random_uuid(),
	season_id uuid not null references seasons(id) on delete cascade,
	number int not null,
	label text not null,
	starts_at timestamptz,
	ends_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint season_weeks_unique_week unique (season_id, number)
);

create table if not exists games (
	id uuid primary key default gen_random_uuid(),
	season_week_id uuid not null references season_weeks(id) on delete cascade,
	game_key text not null unique,
	kickoff timestamptz,
	status text not null default 'scheduled',
	channel text,
	location text,
	home_team jsonb not null,
	away_team jsonb not null,
	home_score int,
	away_score int,
	winner text,
	sportsdata_payload jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists picks (
	id uuid primary key default gen_random_uuid(),
	member_id uuid not null references family_members(id) on delete cascade,
	game_id uuid not null references games(id) on delete cascade,
	chosen_side text not null check (chosen_side in ('home', 'away')),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint picks_unique_member_game unique (member_id, game_id)
);

create table if not exists tie_breakers (
	id uuid primary key default gen_random_uuid(),
	member_id uuid not null references family_members(id) on delete cascade,
	season_week_id uuid not null references season_weeks(id) on delete cascade,
	points int not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint tie_breakers_unique_member_week unique (member_id, season_week_id)
);

create table if not exists week_results (
	id uuid primary key default gen_random_uuid(),
	season_week_id uuid not null unique references season_weeks(id) on delete cascade,
	winner_member_id uuid references family_members(id),
	notes text,
	declared_by_member_id uuid references family_members(id),
	declared_at timestamptz not null default now()
);

create table if not exists season_titles (
	id uuid primary key default gen_random_uuid(),
	season_id uuid not null unique references seasons(id) on delete cascade,
	winner_member_id uuid references family_members(id),
	notes text,
	declared_by_member_id uuid references family_members(id),
	declared_at timestamptz not null default now()
);

create or replace function set_updated_at()
returns trigger as $$
begin
	new.updated_at = now();
	return new;
end;
$$ language plpgsql;

create trigger set_updated_at_seasons
before update on seasons
for each row execute function set_updated_at();

create trigger set_updated_at_season_weeks
before update on season_weeks
for each row execute function set_updated_at();

create trigger set_updated_at_games
before update on games
for each row execute function set_updated_at();

create trigger set_updated_at_picks
before update on picks
for each row execute function set_updated_at();

create trigger set_updated_at_tie_breakers
before update on tie_breakers
for each row execute function set_updated_at();
