# ============================================================
# Multi-stage Dockerfile for the im project
# Stage 1: build the Go binary
# Stage 2: minimal runtime image
# ============================================================

# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder

# Install git (required by some Go modules) and ca-certificates
RUN apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /build

# Cache module downloads: copy go.mod/go.sum first and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a static binary (CGO disabled, since the MySQL driver is pure Go)
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /out/im-server .

# ---------- Runtime stage ----------
FROM alpine:3.20

# Install ca-certificates for HTTPS and tzdata for correct time zone handling
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /out/im-server /app/im-server

# Copy frontend static assets (served by Gin via r.Static("/static", "./frontend"))
COPY frontend /app/frontend

# Copy the DB schema so the container can also initialize MySQL if needed
COPY database/schema.sql /app/database/schema.sql

# Switch to a non-root user for security
USER app

EXPOSE 8080

ENTRYPOINT ["/app/im-server"]
