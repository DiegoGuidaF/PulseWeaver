PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

-- Data loss accepted per ADR 006: original password hashes for demoted users are gone.
-- Backfill a sentinel so NOT NULL constraint can be re-added.
UPDATE users SET password_hash = X'00' WHERE password_hash IS NULL;

CREATE TABLE users_old (
    id                   INTEGER PRIMARY KEY,
    username             TEXT      NOT NULL COLLATE NOCASE,
    display_name         TEXT      NOT NULL,
    email                TEXT      NOT NULL DEFAULT '',
    password_hash        BLOB      NOT NULL,
    role                 TEXT      NOT NULL DEFAULT 'user',
    must_change_password BOOLEAN   NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    created_by           INTEGER   REFERENCES users_old (id) ON DELETE SET NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at           DATETIME
);

INSERT INTO users_old SELECT * FROM users;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
    ON users (username COLLATE NOCASE) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active
    ON users (email) WHERE email != '' AND deleted_at IS NULL;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
