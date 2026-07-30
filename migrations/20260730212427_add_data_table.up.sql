CREATE TABLE IF NOT EXISTS data
(
    id         UUID NOT  NULL PRIMARY KEY,
    user_id    UUID NOT  NULL,
    type       SMALLINT  NOT NULL,
    payload    BYTEA     NOT NULL,
    metadata   TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX data_user_id_idx ON data (user_id);
CREATE INDEX data_type_idx ON data (type);
