.PHONY: run up down restart logs build rebuild clean fresh

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

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
	docker builder prune -f

fresh:
	docker compose down -v --remove-orphans
	docker compose build --no-cache
	docker compose up -d --force-recreate