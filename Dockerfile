# Multi-stage Dockerfile for Zuri Daemon with native pgvector support

# Stage 1: Build Go daemon binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY migrations/ ./migrations/

# Build daemon binary
RUN CGO_ENABLED=0 GOOS=linux go build -o zuri-daemon ./cmd/daemon

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-libc-certificates ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/zuri-daemon /app/zuri-daemon

EXPOSE 7331

ENTRYPOINT ["/app/zuri-daemon"]
