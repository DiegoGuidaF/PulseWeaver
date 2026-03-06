# Stage 1: Frontend Build
FROM node:22-alpine AS frontend-builder

WORKDIR /build

# Copy package files and install dependencies
COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

# Copy API spec file (needed for frontend API type generation)
# The openapi-ts.config.ts references ../api/openapi.yaml relative to frontend/
# So we need api/ at /api/ (one level up from WORKDIR /build)
COPY api/ /api/

# Copy frontend source and build
COPY frontend/ ./
RUN npm run build

# Stage 2: Go Build
FROM golang:1.26-alpine AS backend-builder

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy Go source code
COPY . .

# Copy frontend dist from Stage 1 to internal/ui/dist
COPY --from=frontend-builder /build/dist ./internal/ui/dist

# Build binary with CGO disabled and optimization flags
# CRITICAL: Create /app/data directory in builder stage (distroless has no shell)
# Note: GOOS/GOARCH are auto-detected by Go from Docker's build platform
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -tags=prod \
    -o /app/wallydic \
    ./cmd/api \
    && mkdir -p /app/data

# Stage 3: Final Runtime
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder stage
COPY --from=backend-builder /app/wallydic /app/wallydic

# Copy empty /app/data directory structure from builder stage
# This creates the writable mount point for persistent data
COPY --from=backend-builder /app/data /app/data

# Set default environment variables for data persistence
ENV DB_FILE=/app/data/wallydic.db

# Expose default port
EXPOSE 8080

# Run binary
ENTRYPOINT ["/app/wallydic"]
