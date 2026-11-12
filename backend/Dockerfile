# syntax=docker/dockerfile:1
# check=error=true

# ===========================================
# Edwins Lab — Go Backend Production Dockerfile
# ===========================================
# Builds the Go API from ./cmd/api and runs as non-root.

ARG GO_VERSION=1.26.10
FROM docker.io/library/golang:${GO_VERSION}-bookworm AS build

WORKDIR /app

# Install CA certificates and timezone data (needed at runtime too,
# but we include them in the final stage).
RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

# Cache dependency download layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source and compile statically
COPY . ./

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

# Make binary executable and prepare the upload directory for non-root
RUN chmod 755 /app/edwinslab-api && \
    mkdir -p /app/data/uploads && \
    chown -R edwinslab:edwinslab /app/edwinslab-api /app/data

# Run as non-root
USER 1001:1001

# Listen on the PORT env variable (default 8080)
EXPOSE 8080

ENTRYPOINT ["/app/edwinslab-api"]
