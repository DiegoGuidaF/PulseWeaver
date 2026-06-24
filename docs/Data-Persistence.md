# Data Persistence

The SQLite database is stored at `$DB_DIR/data.db` (defaults to `/data/data.db` in the Docker image).

The database file is plaintext at rest, except for fields PulseWeaver hashes before storing them, such as passwords and
device API keys. It also contains security-sensitive operational data such as host grants, device addresses, and access
logs. Protect the volume with host filesystem permissions, encrypted disks/backups where appropriate, and the same care
you apply to other self-hosted service data.

The official container runs as a non-root user in a distroless runtime and does not include a shell, package manager, or
SQLite CLI. That reduces the tools available to an attacker inside the container, but it is not database encryption and
does not replace host/volume protection: the application process still needs write access to `/data`.

## Docker deployment (recommended)

The image sets `DB_DIR=/data` by default and ships `/data` owned by the container's non-root UID/GID (`65532:65532`).
Mount a writable volume:

- **Named volume (easiest):** `docker run -v pulseweaver-data:/data ...`. Docker initializes a new named volume from
  the image's `/data` directory, so ownership is already correct.
- **Bind mount:** `docker run -v ./data:/data ...`

For a bind mount, Docker uses the host directory exactly as-is; the container cannot `chown` it because the application
runs without root privileges. Ensure the host directory is writable by UID/GID `65532:65532`:

```bash
mkdir -p ./data
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
