CREATE TABLE changes (
    change_id    UUID PRIMARY KEY,
    user_id      UUID NOT NULL,
    device_id    TEXT NOT NULL,

    item_id      UUID NOT NULL,
    operation    TEXT NOT NULL,        -- create | update | delete
    version      INTEGER NOT NULL,

    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
