CREATE TABLE IF NOT EXISTS users
(
    id         UUID         NOT NULL PRIMARY KEY,
    login      VARCHAR(64)  NOT NULL UNIQUE,
    password   VARCHAR(256) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_login_idx ON users (login);
