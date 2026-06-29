.PHONY: run up down restart logs build rebuild clean freshstart

run: up

up:
	docker compose up -d

build:
	docker compose build

rebuild:
	docker compose up -d --build

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

clean:
	docker compose down -v --remove-orphans
	docker image prune -f

fresh:
	docker compose down -v --remove-orphans
	docker rmi gig-platform-core-service-api:latest 2>/dev/null || true
	docker volume rm gig-platform-core-service_postgres_data 2>/dev/null || true
	docker compose up -d --build