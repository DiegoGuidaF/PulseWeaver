CREATE TABLE users
(
    id            INTEGER PRIMARY KEY,
    username      TEXT      NOT NULL UNIQUE COLLATE NOCASE,
    display_name  TEXT      NOT NULL,
    email         TEXT UNIQUE,
    password_hash BLOB      NOT NULL,
    role          TEXT      NOT NULL DEFAULT 'user',
    created_by    INTEGER   REFERENCES users (id) ON DELETE SET NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- browser sessions (server-side)
CREATE TABLE sessions
(
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER REFERENCES users (id) ON DELETE CASCADE,
    token_hash   BLOB     NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   DATETIME NOT NULL,
    last_used_at DATETIME,
    revoked_at   DATETIME
);
CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);