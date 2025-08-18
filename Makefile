.PHONY: build up down stop logs shell build-image

build:
	docker compose build

up:
	docker compose up --build -d

down:
	docker compose down

stop:
	docker compose stop

logs:
	docker compose logs -f

shell:
	docker compose exec api sh

build-image:
	docker build -t kensho-api:local .
