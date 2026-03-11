import { rmSync } from 'node:fs';
import { resolve } from 'node:path';

export function teardown() {
  // Remove the SQLite-backed localStorage file (and its WAL/SHM siblings) that
  // Node 25+ creates when --localstorage-file is passed to the worker processes.
  const base = resolve(process.cwd(), 'tmp-localstorage');
  for (const file of [base, `${base}-shm`, `${base}-wal`]) {
    rmSync(file, { force: true });
  }
}
