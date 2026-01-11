DROP INDEX IF EXISTS idx_ledger_events_type_time;
DROP INDEX IF EXISTS idx_ledger_events_user_time;
DROP INDEX IF EXISTS idx_ledger_events_channel_time;
DROP INDEX IF EXISTS uq_ledger_events_tx_log;

DROP INDEX IF EXISTS uq_payouts_user_chain;

DROP INDEX IF EXISTS idx_social_links_user_platform;
DROP INDEX IF EXISTS uq_social_links_platform_user;

DROP INDEX IF EXISTS idx_identities_user_id;
DROP INDEX IF EXISTS uq_identities_provider_user;

DROP TABLE IF EXISTS ledger_events;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS social_links;
DROP TABLE IF EXISTS identities;
DROP TABLE IF EXISTS users;
