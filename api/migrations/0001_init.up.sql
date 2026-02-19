CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  login TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tournaments (
  id BIGSERIAL PRIMARY KEY,
  game TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT,
  start_date DATE,
  end_date DATE,
  allow_odd BOOLEAN NOT NULL DEFAULT TRUE,
  status TEXT NOT NULL DEFAULT 'DRAFT',
  is_listed BOOLEAN NOT NULL DEFAULT FALSE,
  is_bracket_published BOOLEAN NOT NULL DEFAULT FALSE,
  schedule_visibility_ahead TEXT NOT NULL DEFAULT '0',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS teams (
  id BIGSERIAL PRIMARY KEY,
  tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  note TEXT,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  seed INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tournament_id, name)
);

CREATE TABLE IF NOT EXISTS matches (
  id BIGSERIAL PRIMARY KEY,
  tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
  round INTEGER NOT NULL,
  index_in_round INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'SCHEDULED',
  bo INTEGER NOT NULL DEFAULT 1,
  dispute_rule TEXT NOT NULL DEFAULT 'ADMIN_DECISION',
  team_a_id BIGINT REFERENCES teams(id),
  team_b_id BIGINT REFERENCES teams(id),
  score_a INTEGER NOT NULL DEFAULT 0,
  score_b INTEGER NOT NULL DEFAULT 0,
  winner_team_id BIGINT REFERENCES teams(id),
  side_assignment_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  starts_at TIMESTAMPTZ,
  ended_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tournament_id, round, index_in_round)
);

CREATE TABLE IF NOT EXISTS schedule (
  id BIGSERIAL PRIMARY KEY,
  tournament_id BIGINT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
  match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  position INTEGER NOT NULL,
  UNIQUE (tournament_id, position),
  UNIQUE (tournament_id, match_id)
);

CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT REFERENCES users(id),
  tournament_id BIGINT REFERENCES tournaments(id) ON DELETE CASCADE,
  entity TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  action TEXT NOT NULL,
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tournaments_game ON tournaments(game);
CREATE INDEX IF NOT EXISTS idx_tournaments_status ON tournaments(status);
CREATE INDEX IF NOT EXISTS idx_tournaments_is_listed ON tournaments(is_listed);
CREATE INDEX IF NOT EXISTS idx_matches_tournament_round ON matches(tournament_id, round);
CREATE INDEX IF NOT EXISTS idx_schedule_tournament_position ON schedule(tournament_id, position);
CREATE INDEX IF NOT EXISTS idx_audit_tournament ON audit_log(tournament_id);
