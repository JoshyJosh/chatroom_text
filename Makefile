.PHONY: run test

run:
	go run cmd/main.go

test:
	go test ./...

up:
	docker compose up -d

down:
	docker compose down