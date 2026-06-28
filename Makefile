
.PHONY: run up down restart logs build rebuild clean

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

