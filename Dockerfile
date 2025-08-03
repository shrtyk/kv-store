FROM golang:1.24.5-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/kv-store -ldflags="-w -s" ./cmd/app

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/kv-store .
COPY config/config.yml ./config/config.yml
RUN mkdir -p ./data/wal ./data/snapshots

EXPOSE 16700

ENTRYPOINT ["/app/kv-store", "-cfg_path=./config/config.yml"]
