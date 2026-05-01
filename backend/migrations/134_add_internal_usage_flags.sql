-- Track internal/admin usage separately from customer profit.
-- Internal usage is still counted and costed, but excluded from customer profit.

ALTER TABLE users ADD COLUMN IF NOT EXISTS internal_usage BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS internal_usage BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN users.internal_usage IS 'When true, this user usage is tracked as internal cost and excluded from customer profit.';
COMMENT ON COLUMN api_keys.internal_usage IS 'When true, this API key usage is tracked as internal cost and excluded from customer profit.';
