.PHONY: run test up down build certs help

# NOTE:
# filter-out commands allows for adding arguments to the end of targets

run:
	(cd app && go run cmd/main.go) && cd ..

test:
	(cd app && go test ./... --count 1 --race) && cd ..

up:
	docker compose up -d $(filter-out $@,$(MAKECMDGOALS))

down:
	docker compose down $(filter-out $@,$(MAKECMDGOALS))

build:
	docker compose build

logs:
	docker compose logs -f $(filter-out $@,$(MAKECMDGOALS))

# generate tls certs
certs:
	./generate.sh

help:
	@echo "Available targets:"
	@echo "  run           : Run the application"
	@echo "  test          : Run tests"
	@echo "  up   [service]: Start containers (e.g., 'make up chatroom'), defaults to all services"
	@echo "  down [service]: Stop containers (e.g., 'make down chatroom'), defaults to all services"
	@echo "  build         : Build chatroom container"
	@echo "  logs [service]: Show logs for services (e.g., 'make logs chatroom'), defaults to all services"
	@echo "  certs         : Generate TLS certificates"
	@echo "  help          : Show this help message"