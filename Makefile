.PHONY: build up down logs test seed migrate

build:
	docker compose build

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f api worker

migrate:
	docker compose run --rm migrate

seed:
	docker compose exec -T postgres psql -U meta -d meta_gateway -f /seed/seed.sql

test:
	docker compose run --rm api go test ./...
