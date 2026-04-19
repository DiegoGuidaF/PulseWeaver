PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE users_new (
    id                   INTEGER PRIMARY KEY,
    username             TEXT      NOT NULL COLLATE NOCASE,
    display_name         TEXT      NOT NULL,
    email                TEXT      NOT NULL DEFAULT '',
    password_hash        BLOB,
    role                 TEXT      NOT NULL DEFAULT 'user',
    must_change_password BOOLEAN   NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    created_by           INTEGER   REFERENCES users_new (id) ON DELETE SET NULL,
    created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at           DATETIME
);

INSERT INTO users_new SELECT * FROM users;

UPDATE sessions
SET revoked_at = CURRENT_TIMESTAMP
WHERE user_id IN (SELECT id FROM users_new WHERE role = 'user')
  AND revoked_at IS NULL;

UPDATE users_new SET password_hash = NULL WHERE role = 'user';

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_active
    ON users (username COLLATE NOCASE) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active
    ON users (email) WHERE email != '' AND deleted_at IS NULL;

PRAGMA foreign_key_check;
COMMIT;
PRAGMA foreign_keys = ON;
