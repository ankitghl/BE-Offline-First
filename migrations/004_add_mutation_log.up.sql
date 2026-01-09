CREATE TABLE IF NOT EXISTS mutation_log (
    mutation_id UUID PRIMARY KEY,
    item_id UUID NOT NULL,
    mutation_type TEXT NOT NULL CHECK (mutation_type IN ('create', 'update', 'delete')),
    applied_version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mutation_log_item_id
ON mutation_log(item_id);