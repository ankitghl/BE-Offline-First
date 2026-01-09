-- Global sync version table
CREATE TABLE IF NOT EXISTS sync_state (
  id INT PRIMARY KEY CHECK (id = 1),
  latest_version BIGINT NOT NULL
);

-- Initialize global version counter
INSERT INTO sync_state (id, latest_version)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;
