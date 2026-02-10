CREATE TABLE device_tokens (
                               id TEXT PRIMARY KEY,
                               user_id TEXT NOT NULL,
                               device_id TEXT,                   -- optional: bind token to a device record
                               label TEXT NOT NULL,              -- e.g. "Pixel 8", "RaspberryPi"
                               token_prefix TEXT NOT NULL,       -- for lookup without storing raw token
                               token_hash BLOB NOT NULL,         -- store hash/HMAC(secret)
                               created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                               last_used_at TIMESTAMP,
                               revoked_at TIMESTAMP,
                               FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_device_tokens_prefix ON device_tokens(token_prefix);
