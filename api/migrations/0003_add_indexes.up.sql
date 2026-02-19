CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_tournament_status ON matches(tournament_id, status);
CREATE INDEX IF NOT EXISTS idx_matches_starts_at ON matches(starts_at);
