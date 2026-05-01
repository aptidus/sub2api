-- Commercial billing hardening:
-- 1. map Sub2API subscription plans to Stripe recurring Price IDs
-- 2. speed up reconciliation between idempotent billing rows and usage logs

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS stripe_price_id VARCHAR(128) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_subscription_plans_stripe_price_id
    ON subscription_plans (stripe_price_id)
    WHERE stripe_price_id <> '';

CREATE INDEX IF NOT EXISTS idx_usage_logs_request_api_key
    ON usage_logs (request_id, api_key_id)
    WHERE request_id IS NOT NULL AND request_id <> '';
