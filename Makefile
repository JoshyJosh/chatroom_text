.PHONY: run test

run:
	go run cmd/main.go

test:
	go test ./...