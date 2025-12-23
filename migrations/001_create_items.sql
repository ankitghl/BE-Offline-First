CREATE TABLE items (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,

    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,

    version     INTEGER NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    deleted_at  TIMESTAMPTZ,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
