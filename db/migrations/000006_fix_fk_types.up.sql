-- Fix user_id column types (should be bigint, not bigserial/sequence)
ALTER TABLE identities ALTER COLUMN user_id TYPE bigint;
ALTER TABLE social_links ALTER COLUMN user_id TYPE bigint;
ALTER TABLE payouts ALTER COLUMN user_id TYPE bigint;
ALTER TABLE ledger_events ALTER COLUMN user_id TYPE bigint;

-- Cleanup: drop schemas effectively created by bigserial (if any)
-- Postgres usually doesn't create multiple sequences if they were defined in one block, 
-- but converting them to TYPE bigint is safer for FK relations.
