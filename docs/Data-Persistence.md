# Data Persistence

SQLite database is stored at `$DB_DIR/data.db` (defaults to `/data/data.db` in the Docker image).

## Docker deployment (recommended)

The image sets `DB_DIR=/data` by default. Mount a writable volume:

- **Named volume (easiest):** `docker run -v pulseweaver-data:/data ...`
- **Bind mount:** `docker run -v ./data:/data ...`

Docker runs as non-root UID/GID `65532:65532` (`gcr.io/distroless/static-debian12:nonroot`), so ensure `/data` is
writable:

```bash
sudo chown -R 65532:65532 ./data
```

## Local development

Defaults to `./data/data.db`. No config needed.

## Custom path

Override with `DB_DIR`:

```bash
DB_DIR=/custom/path go run cmd/api/main.go
# or
docker run -e DB_DIR=/custom/path ...
```

## SQLite WAL mode warning

Shared-memory coordination (`data.db-wal`, `data.db-shm`) requires local disk — not NFS/SMB. For backups, include
WAL/SHM files or run `PRAGMA wal_checkpoint(FULL)` first.
