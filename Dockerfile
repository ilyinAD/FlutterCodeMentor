# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go tool oapi-codegen -config api/config.yaml api/openapi.yaml

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/app ./cmd


FROM alpine:3.20

RUN apk add --no-cache \
    git \
    docker-cli \
    ca-certificates \
    tzdata \
    wget

WORKDIR /app

COPY --from=builder /out/app /app/app

RUN mkdir -p /app/repos /app/snippets

ENV STORAGE_REPOS_DIR=/app/repos \
    STORAGE_SNIPPETS_DIR=/app/snippets \
    SERVER_PORT=8080

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/app"]
