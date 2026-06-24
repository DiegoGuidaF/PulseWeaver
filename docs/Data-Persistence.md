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

## Backing up

It's one SQLite database, so a backup is one file — but PulseWeaver runs in WAL mode, where recent writes may live in
the `data.db-wal` sidecar rather than in `data.db` itself. Don't copy `data.db` alone while the app is running, or you
may capture a stale snapshot. Pick one of:

- **Online, consistent (recommended)** — let SQLite produce a single coherent file with no downtime:
  ```bash
  sqlite3 /path/to/data/data.db ".backup '/path/to/backup/data.db'"
  ```
  The result is a standalone `data.db` with the WAL already folded in — copy it anywhere.

- **Checkpoint then copy** — fold the WAL back into the main file, then copy it:
  ```bash
  sqlite3 /path/to/data/data.db "PRAGMA wal_checkpoint(FULL);"
  cp /path/to/data/data.db /path/to/backup/data.db
  ```

- **Copy everything** — if you stop the container first, copying the whole `data/` directory (including `data.db-wal`
  and `data.db-shm`) is also safe.

To restore, stop PulseWeaver, drop the backup file in place as `$DB_DIR/data.db` (remove any stale `-wal`/`-shm`
sidecars), and start it again.

## SQLite WAL mode note

Shared-memory coordination (`data.db-wal`, `data.db-shm`) requires local disk — **not** NFS/SMB. Keep the data
directory on a local volume.
