# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o discord-bot \
    ./cmd/bot

# Runtime stage
FROM gcr.io/distroless/base:latest

COPY --from=builder /app/discord-bot /usr/local/bin/discord-bot

CMD ["discord-bot"]
