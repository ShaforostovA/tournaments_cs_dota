SHELL := /bin/sh

.PHONY: up down logs migrate seed

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

migrate:
	docker compose run --rm migrate

seed:
	docker compose run --rm api go run ./cmd/seed
