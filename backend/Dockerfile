# Stage 1: Build the Go binary
FROM golang:1.26-alpine AS builder

# Git is needed by some go modules
RUN apk add --no-cache git

WORKDIR /app

# Cache dependency downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: modernc.org/sqlite is pure Go — no C toolchain needed
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/astroledger ./cmd/server

# Stage 2: Minimal runtime image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/astroledger .

# Database and migrations are embedded/applied at startup
# Mount a volume at /data for the SQLite file
RUN mkdir -p /data

EXPOSE 8080

ENTRYPOINT ["./astroledger"]
