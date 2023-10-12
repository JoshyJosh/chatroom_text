FROM golang:1.21.3-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /bin/chatroom ./cmd/main.go

FROM builder as runner
COPY --from=builder /bin/chatroom /bin/
# copy local certs for TLS server
COPY key.pem key.pem
COPY cert.pem cert.pem
CMD ["/bin/chatroom"]