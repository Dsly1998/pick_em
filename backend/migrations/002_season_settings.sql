create table if not exists season_settings (
	season_id uuid primary key references seasons(id) on delete cascade,
	current_week int not null default 1 check (current_week > 0),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create trigger set_updated_at_season_settings
before update on season_settings
for each row execute function set_updated_at();

