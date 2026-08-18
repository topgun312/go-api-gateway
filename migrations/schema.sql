CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    api_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (api_key, name) VALUES ('test-key-123', 'test user')
ON CONFLICT (api_key) DO NOTHING;