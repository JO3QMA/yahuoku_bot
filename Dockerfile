# syntax=docker/dockerfile:1
# Build stage (with BuildKit cache)
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o discord-bot \
    ./cmd/bot

# Runtime stage (distroless:noroot)
# WORKDIR /app にしておくことで DB_PATH=data/watch.db が /app/data/watch.db に解決され、
# compose の bot-data ボリューム（/app/data）に DB が作成される。
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/discord-bot /usr/local/bin/discord-bot

ENTRYPOINT ["/usr/local/bin/discord-bot"]
