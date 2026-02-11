ALTER TABLE users
    ADD COLUMN role TEXT DEFAULT 'user';

ALTER TABLE users
    ADD COLUMN created_by INTEGER
        REFERENCES users (id)
            ON DELETE SET NULL;
