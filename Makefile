.PHONY: run test up down build

run:
	go run cmd/main.go

test:
	go test ./...

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build

# filter-out command allows for adding service names at 
# the end of the docker compose logs command
logs:
	docker compose logs -f $(filter-out $@,$(MAKECMDGOALS))