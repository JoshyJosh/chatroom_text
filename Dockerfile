FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY app/ .
RUN go build -o /bin/chatroom ./cmd/main.go

FROM builder AS runner
COPY --from=builder /bin/chatroom /bin/
CMD ["/bin/chatroom"]

FROM golang:1.22-alpine AS dev-runner
WORKDIR /app
# RUN go mod download -x
RUN go install github.com/githubnemo/CompileDaemon@latest
ENTRYPOINT CompileDaemon --build="go build cmd/main.go" --command="./main" --directory="/app"