# Docker Deployment Guide

This guide covers building and running WallyDic using Docker and Docker Compose.

## Overview

WallyDic uses a multi-stage Docker build process to create a minimal, production-ready container image:

1. **Frontend Build Stage**: Builds the React frontend using Node.js
2. **Go Build Stage**: Compiles the Go backend and embeds the frontend
3. **Runtime Stage**: Uses `distroless/static` for a minimal, secure final image

The final image is approximately 30-50MB and contains only the compiled binary with no shell or unnecessary tools.

## Prerequisites

- Docker 20.10+ (with BuildKit support)
- Docker Compose 2.0+ (optional, for docker-compose.yml)

## Building the Image

### Basic Build

```bash
docker build -t wallydic:latest .
```

### Build with Tag

```bash
docker build -t wallydic:v1.0.0 .
```

### Build Arguments

The Dockerfile doesn't currently accept build arguments, but you can override environment variables at runtime (see [Environment Variables](#environment-variables)).

## Running the Container

### Using Docker Run

#### Basic Run (with named volume)

```bash
docker run -d \
  --name wallydic \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secure-password \
  -v wallydic-data:/app/data \
  wallydic:latest
```

#### Development Run (with bind mount)

```bash
docker run -d \
  --name wallydic \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your-secure-password \
  -v ./data:/app/data \
  wallydic:latest
```

#### Custom Port

```bash
docker run -d \
  --name wallydic \
  -p 9090:8080 \
  -e ADMIN_PASSWORD=your-secure-password \
  -e SERVER_PORT=8080 \
  -v wallydic-data:/app/data \
  wallydic:latest
```

### Using Docker Compose

#### Basic Usage

1. Create a `.env` file (optional, for environment variables):

```bash
ADMIN_PASSWORD=your-secure-password
LOG_LEVEL=info
TZ=UTC
```

2. Start the service:

```bash
docker-compose up -d
```

3. View logs:

```bash
docker-compose logs -f wallydic
```

4. Stop the service:

```bash
docker-compose down
```

#### Development with Bind Mount

To use a local directory instead of a Docker volume, modify `docker-compose.yml`:

```yaml
volumes:
  - ./data:/app/data  # Instead of wallydic-data:/app/data
```

#### Rebuild After Code Changes

```bash
docker-compose build --no-cache
docker-compose up -d
```

## Environment Variables

### Required Variables

- **`ADMIN_PASSWORD`**: Password for the admin user (required by application config)

### Optional Variables

All optional variables have defaults set in the Dockerfile or application config:

- **`SERVER_PORT`**: HTTP server port (default: `8080`)
- **`DB_FILE`**: Database file path (default: `/app/data/wallydic.db`)
- **`WHITELIST_FILE_PATH`**: Whitelist file path (default: `/app/data/whitelist.txt`)
- **`LOG_LEVEL`**: Logging level - `debug`, `info`, `warn`, `error` (default: `info`)
- **`LOG_FORMAT`**: Log format - `json` or `text` (default: `text`)
- **`LOG_COLOR`**: Enable colored output for tint format - `true` or `false` (default: `false`). Ignored when `LOG_FORMAT=json`
- **`TZ`**: Timezone (default: `UTC`)
- **`TRUSTED_PROXY`**: Trusted proxy IP/CIDR (default: empty)
- **`WHITELIST_DEBOUNCE_DELAY`**: Whitelist regeneration debounce delay (default: `5s`)
- **`DB_DEBUG`**: Enable SQL debug logging (default: `false`)

### Setting Environment Variables

#### Docker Run

```bash
docker run -d \
  -e ADMIN_PASSWORD=secret \
  -e LOG_LEVEL=debug \
  -e TZ=America/New_York \
  wallydic:latest
```

#### Docker Compose

In `docker-compose.yml`:

```yaml
environment:
  - ADMIN_PASSWORD=secret
  - LOG_LEVEL=debug
  - TZ=America/New_York
```

Or use a `.env` file:

```bash
ADMIN_PASSWORD=secret
LOG_LEVEL=debug
TZ=America/New_York
```

## Volume Mounts

### Persistent Storage

WallyDic stores data in `/app/data`:

- **Database**: `/app/data/wallydic.db` (SQLite database)
- **Whitelist**: `/app/data/whitelist.txt` (generated whitelist file)

### Volume Options

#### Named Volume (Recommended for Production)

```bash
docker run -d \
  -v wallydic-data:/app/data \
  wallydic:latest
```

Create and manage the volume:

```bash
# Create volume
docker volume create wallydic-data

# Inspect volume
docker volume inspect wallydic-data

# Remove volume (WARNING: deletes all data)
docker volume rm wallydic-data
```

#### Bind Mount (Recommended for Development)

```bash
docker run -d \
  -v ./data:/app/data \
  wallydic:latest
```

This mounts a local `./data` directory, making it easy to inspect and backup files.

#### Docker Compose Volume

The `docker-compose.yml` uses a named volume by default:

```yaml
volumes:
  wallydic-data:
    driver: local
```

To use a bind mount instead, change to:

```yaml
volumes:
  - ./data:/app/data
```

## Health Checks

The `docker-compose.yml` includes a health check that verifies the application is responding:

```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

**Note**: The distroless image doesn't include `wget` or `curl`, so the health check in docker-compose.yml may not work. Consider using an external health check or monitoring tool instead.

## Networking

### Port Mapping

The default port is `8080`. Map it to a different host port:

```bash
docker run -d -p 9090:8080 wallydic:latest
```

### Docker Network

Create a custom network for isolation:

```bash
docker network create wallydic-net

docker run -d \
  --network wallydic-net \
  --name wallydic \
  wallydic:latest
```

## Troubleshooting

### Container Won't Start

1. **Check logs**:

```bash
docker logs wallydic
```

2. **Verify environment variables**:

```bash
docker inspect wallydic | grep -A 20 Env
```

3. **Check volume permissions**:

The distroless image runs as a non-root user (uid 65532). Ensure the mounted volume is writable:

```bash
chmod 777 ./data  # Development only
# Or use proper ownership for production
```

### Database Not Persisting

- Verify the volume is mounted: `docker inspect wallydic | grep Mounts`
- Check volume permissions (see above)
- Ensure `DB_FILE` environment variable points to `/app/data/wallydic.db`

### Frontend Not Loading

- Verify the build completed successfully (check Docker build logs)
- Check that the frontend was embedded: `docker run --rm wallydic:latest ls -la /app/`
- Ensure you're accessing the correct port

### Build Failures

1. **Clear build cache**:

```bash
docker build --no-cache -t wallydic:latest .
```

2. **Check BuildKit**:

```bash
DOCKER_BUILDKIT=1 docker build -t wallydic:latest .
```

3. **Verify dependencies**:

- Node.js 22+ for frontend build
- Go 1.26+ for backend build

## Image Details

### Base Images

- **Frontend Builder**: `node:22-alpine`
- **Backend Builder**: `golang:1.26-alpine`
- **Runtime**: `gcr.io/distroless/static-debian12:nonroot`

### Image Size

- Final image: ~30-50MB (depends on Go binary size)
- Build stages: ~500MB+ (not included in final image)

### Security

- **Non-root user**: Runs as uid 65532 (distroless nonroot user)
- **No shell**: Distroless image has no shell, reducing attack surface
- **Minimal base**: Only contains the binary and essential runtime libraries
- **CGO disabled**: Pure Go binary, no C dependencies

## Production Considerations

### Resource Limits

Set resource limits in `docker-compose.yml`:

```yaml
services:
  wallydic:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Logging

For production, use JSON logging:

```yaml
environment:
  - LOG_FORMAT=json
```

### Backup Strategy

Backup the `/app/data` volume regularly:

```bash
# Backup database
docker run --rm \
  -v wallydic-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/wallydic-backup-$(date +%Y%m%d).tar.gz -C /data .
```

### Updates

1. Pull/build new image
2. Stop container: `docker-compose down`
3. Start with new image: `docker-compose up -d`
4. Database migrations run automatically on startup

## Examples

### Complete Production Setup

```bash
# Build image
docker build -t wallydic:v1.0.0 .

# Create volume
docker volume create wallydic-data

# Run container
docker run -d \
  --name wallydic \
  --restart unless-stopped \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=$(openssl rand -base64 32) \
  -e LOG_FORMAT=json \
  -e TZ=UTC \
  -v wallydic-data:/app/data \
  wallydic:v1.0.0
```

### Development Setup

```bash
# Create local data directory
mkdir -p ./data

# Run with bind mount
docker run -d \
  --name wallydic-dev \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=dev-password \
  -e LOG_LEVEL=debug \
  -v ./data:/app/data \
  wallydic:latest

# View logs
docker logs -f wallydic-dev

# Access shell (if needed, use alpine image for debugging)
docker run --rm -it --entrypoint sh alpine:latest
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Distroless Images](https://github.com/GoogleContainerTools/distroless)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
