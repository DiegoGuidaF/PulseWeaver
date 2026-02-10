-- users
CREATE TABLE users
(
    id            INTEGER PRIMARY KEY,
    name          TEXT      NOT NULL,
    email         TEXT      NOT NULL UNIQUE,
    password_hash BLOB      NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- browser sessions (server-side)
CREATE TABLE sessions
(
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER  NOT NULL,
    token_hash   BLOB     NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   DATETIME NOT NULL,
    last_used_at DATETIME,
    revoked_at   DATETIME,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);