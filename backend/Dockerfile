# Stage 1: Build the SvelteKit static frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --prefer-offline

COPY frontend/ ./
RUN npm run build

# Stage 2: Build the Go binary (with embedded frontend)
FROM golang:1.26-alpine AS builder

# Git is needed by some go modules
RUN apk add --no-cache git

WORKDIR /app

# Cache dependency downloads separately from source
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

# Overlay the real frontend build into the embed directory so //go:embed picks it up.
COPY --from=frontend-builder /frontend/build ./internal/frontend/dist/

# CGO_ENABLED=0: modernc.org/sqlite is pure Go — no C toolchain needed
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/astroledger ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/astroledger .

# Database and migrations are embedded/applied at startup
# Mount a volume at /data for the SQLite file
RUN mkdir -p /data

EXPOSE 8080

ENTRYPOINT ["./astroledger"]
