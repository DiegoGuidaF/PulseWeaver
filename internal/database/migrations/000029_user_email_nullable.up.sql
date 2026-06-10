PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

-- SQLite cannot drop a NOT NULL constraint via ALTER TABLE.
-- Use the create-copy-drop-rename pattern (foreign_keys must be OFF to
-- prevent child-table FK references from following the rename).
CREATE TABLE users_new
(
    id                   INTEGER PRIMARY KEY,
    username             TEXT      NOT NULL COLLATE NOCASE,
    display_name         TEXT      NOT NULL,
    email                TEXT,
    password_hash        BLOB,
    role                 TEXT      NOT NULL DEFAULT 'user',
    must_change_password BOOLEAN   NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    created_by           INTEGER   REFERENCES users_new (id) ON DELETE SET NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at           DATETIME
);

INSERT INTO users_new (id, username, display_name, email, password_hash, role,
                       must_change_password, created_by, created_at, deleted_at)
    SELECT id, username, display_name, email, password_hash, role,
           must_change_password, created_by, created_at, deleted_at
    FROM users;

DROP TABLE users;

ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX idx_users_username_active
    ON users (username COLLATE NOCASE) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_users_email_active
    ON users (email) WHERE email != '' AND deleted_at IS NULL;

PRAGMA foreign_key_check;

COMMIT;

PRAGMA foreign_keys = ON;
