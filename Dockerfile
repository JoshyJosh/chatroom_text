FROM golang:1.21.3-alpine AS builder
WORKDIR /app
COPY app/ .
RUN go build -o /bin/chatroom ./cmd/main.go

FROM builder as runner
COPY --from=builder /bin/chatroom /bin/
# copy local certs for TLS server
COPY certs/key.pem key.pem
COPY certs/cert.pem cert.pem
CMD ["/bin/chatroom"]