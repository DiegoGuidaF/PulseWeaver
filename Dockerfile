# syntax=docker/dockerfile:1

# Stage 1: Frontend Build
FROM --platform=$BUILDPLATFORM node:25.8-alpine AS frontend-builder

WORKDIR /app

# Install root-level deps (redocly for API bundling)
COPY package*.json ./
RUN --mount=type=cache,id=npm-root,target=/root/.npm,sharing=locked \
    npm ci

# Copy split API spec files (bundled during npm run build via pregenerate:api)
COPY api/ ./api/

# Install frontend deps
COPY frontend/package*.json ./frontend/
RUN --mount=type=cache,id=npm-frontend,target=/root/.npm,sharing=locked \
    npm ci --prefix frontend

# Build: prebuild → generate:api → pregenerate:api → bundle:api → openapi-ts → tsc + vite
COPY frontend/ ./frontend/
RUN npm run build --prefix frontend

# Stage 2: Go Build
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS backend-builder

WORKDIR /build

ARG TARGETOS
ARG TARGETARCH

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod,sharing=locked \
    go mod download

# Copy Go source code
COPY . .

# Bundle is generated inside the frontend stage — bring it in for go:generate/go:embed
COPY --from=frontend-builder /app/api/openapi-bundle.gen.yaml ./api/openapi-bundle.gen.yaml

# Copy frontend dist from Stage 1 to internal/ui/dist
COPY --from=frontend-builder /app/frontend/dist ./internal/ui/dist

# Build tags: prod by default, so the released image (release.yml / a plain
# `docker build`) is byte-for-byte the production binary with no debug surface.
# Pass GO_TAGS="prod pprof" to compile in the loopback pprof debug listener
# (127.0.0.1:6060) for a profiling build — see prod-deployment's pprof profile.
ARG GO_TAGS=prod

# Build binary with CGO disabled and optimization flags
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=gobuild-${TARGETOS}-${TARGETARCH},target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w" \
    -tags="$GO_TAGS" \
    -o /app/pulseweaver \
    ./cmd/api

RUN mkdir -p /runtime-data/geoip

# Stage 3: Final Runtime
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy binary from builder stage
COPY --from=backend-builder /app/pulseweaver /app/pulseweaver
COPY --from=backend-builder --chown=65532:65532 /runtime-data /data

# Mount a writable volume at /data — see README for bind mount ownership requirements (UID/GID 65532:65532).
ENV DB_DIR=/data
ENV GEOIP_DATA_DIR=/data/geoip

# Expose default port
EXPOSE 8080

# Run binary
ENTRYPOINT ["/app/pulseweaver"]
