CREATE TYPE file_status AS ENUM ('IN_PROGRESS', 'COMPLETED', 'FAILED');

CREATE TABLE file (
    id           UUID          NOT NULL PRIMARY KEY,
    user_id      UUID          NOT NULL,
    name         VARCHAR(1024) NOT NULL,
    mime_type    VARCHAR(256)  NOT NULL,
    size         BIGINT        NOT NULL,
    chunks_count INT           NOT NULL,
    status       file_status   NOT NULL DEFAULT 'IN_PROGRESS',
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT   fk_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE file_chunk (
    id          UUID    NOT NULL PRIMARY KEY,
    file_id     UUID    NOT NULL,
    chunk_index INT     NOT NULL,
    uploaded    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT  fk_file_id FOREIGN KEY (file_id) REFERENCES file (id) ON DELETE CASCADE,
    UNIQUE (file_id, chunk_index)
);

CREATE INDEX file_user_id_idx ON file (user_id);
CREATE INDEX file_chunk_file_id_idx ON file_chunk (file_id);
