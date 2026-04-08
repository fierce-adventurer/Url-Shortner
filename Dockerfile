FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o shortener ./cmd/api

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/shortener .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./shortener"]