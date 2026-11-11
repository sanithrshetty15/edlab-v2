# syntax=docker/dockerfile:1
# check=error=true

# ===========================================
# Edwins Lab — Go Backend Production Dockerfile
# ===========================================
# Replaces the Ruby on Rails Dockerfile.
# Builds the Go API from ./cmd/api and runs as non-root.

ARG GO_VERSION=1.25.0
FROM docker.io/library/golang:${GO_VERSION}-bookworm AS build

WORKDIR /app

# Install CA certificates and timezone data (needed at runtime too,
# but we include them in the final stage). Also install git for go get.
RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y ca-certificates tzdata git && \
    rm -rf /var/lib/apt/lists/*

# Cache dependency download layer
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source and compile statically
COPY backend/ ./

# Build the Go API binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -extldflags '-static'" \
    -o /edwinslab-api ./cmd/api

# ---- Runtime image ----
FROM docker.io/library/debian:bookworm-slim AS runtime

RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

# Create a non-root user
RUN groupadd --system --gid 1001 edwinslab && \
    useradd --system --uid 1001 --gid 1001 --no-create-home --shell /usr/sbin/nologin edwinslab

WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /edwinslab-api /app/edwinslab-api

# Make binary executable and owned by non-root
RUN chmod 755 /app/edwinslab-api && \
    chown edwinslab:edwinslab /app/edwinslab-api

# Run as non-root
USER 1001:1001

# Listen on the PORT env variable (default 8080)
EXPOSE 8080

ENTRYPOINT ["/app/edwinslab-api"]
