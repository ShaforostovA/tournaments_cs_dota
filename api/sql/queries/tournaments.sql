-- name: ListTournaments :many
SELECT
  id,
  game,
  name,
  description,
  start_date,
  end_date,
  status,
  is_listed,
  is_bracket_published,
  schedule_visibility_ahead,
  created_at,
  updated_at
FROM tournaments
WHERE ($1::text = '' OR game = $1)
ORDER BY created_at DESC;
