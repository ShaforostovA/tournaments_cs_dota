# Tournaments CS2 / Dota 2

Monorepo for a tournament bracket service.

## Quick start (dev)

```bash
make up
```

This starts Postgres via Docker Compose.

## Commands

- `make up` – start services
- `make down` – stop services
- `make logs` – follow compose logs
- `make migrate` – run DB migrations (placeholder for now)
- `make seed` – seed dev data (placeholder for now)

## Audit Log

- Admin UI: open any tournament in `/admin` and scroll to the `Audit log` section.
- API: `GET /api/v1/admin/tournaments/{id}/audit?page=&pageSize=` (requires admin session).

## Healthchecks (Docker Compose)

- `db`: `pg_isready -U tournaments -d tournaments`
- `api`: `GET /health`
- `web`: `GET /`

## Production-relevant env vars

Backend:
- `CORS_ALLOWED_ORIGINS` – comma-separated allowlist (e.g. `https://example.com,https://admin.example.com`)
- `COOKIE_SECURE` – set `true` in production to require HTTPS cookies
- `COOKIE_SAMESITE` – `Lax` (default), `Strict`, or `None`
- `LOGIN_RATE_LIMIT` – max login attempts per window (default `5`)
- `LOGIN_RATE_WINDOW` – duration for login rate window (default `1m`)

Frontend:
- `NEXT_PUBLIC_API_URL` – API base URL for browser requests

## Structure

- `api/` Go backend (coming next)
- `web/` Next.js frontend (coming next)
- `infra/` infrastructure helpers
- `docs/` docs and progress
