FROM golang:1.21.3-alpine AS builder
WORKDIR /app
COPY app/ .
RUN go build -o /bin/chatroom ./cmd/main.go

FROM builder as runner
COPY --from=builder /bin/chatroom /bin/
CMD ["/bin/chatroom"]