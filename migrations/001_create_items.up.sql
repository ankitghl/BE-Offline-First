CREATE TABLE IF NOT EXISTS items (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,

    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,

    version     INTEGER NOT NULL,
    deleted     BOOLEAN NOT NULL DEFAULT FALSE,

    updated_at  TIMESTAMP NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT now()
);
