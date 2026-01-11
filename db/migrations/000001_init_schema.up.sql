CREATE TABLE "users" (
  "id" bigserial PRIMARY KEY,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE "identities" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "provider" varchar NOT NULL,
  "provider_user_id" varchar NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE "social_links" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "platform" varchar NOT NULL,
  "platform_user_id" varchar NOT NULL,
  "verified_at" timestamp,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE "payouts" (
  "id" bigserial PRIMARY KEY,
  "user_id" bigserial NOT NULL,
  "chain" varchar NOT NULL,
  "address" varchar NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE TABLE "ledger_events" (
  "id" bigserial PRIMARY KEY,
  "platform" varchar NOT NULL,
  "platform_user_id" varchar NOT NULL,
  "user_id" bigserial,
  "event_type" varchar NOT NULL,
  "amount_raw" numeric(78,0) NOT NULL,
  "message" text,
  "tx_hash" varchar NOT NULL,
  "log_index" int NOT NULL,
  "block_time" timestamp NOT NULL,
  "created_at" timestamp NOT NULL,
  "updated_at" timestamp NOT NULL
);

CREATE UNIQUE INDEX ON "identities" ("provider", "provider_user_id");

CREATE INDEX ON "identities" ("user_id");

CREATE UNIQUE INDEX ON "social_links" ("platform", "platform_user_id");

CREATE INDEX ON "social_links" ("user_id", "platform");

CREATE UNIQUE INDEX ON "payouts" ("user_id", "chain");

CREATE UNIQUE INDEX ON "ledger_events" ("tx_hash", "log_index");

CREATE INDEX ON "ledger_events" ("platform", "platform_user_id", "block_time");

CREATE INDEX ON "ledger_events" ("user_id", "block_time");

CREATE INDEX ON "ledger_events" ("event_type", "block_time");

COMMENT ON COLUMN "identities"."provider" IS '''google'' | ''wallet''';

COMMENT ON COLUMN "identities"."provider_user_id" IS 'google: sub; wallet: 0x... lowercased';

COMMENT ON COLUMN "social_links"."platform" IS '''youtube''';

COMMENT ON COLUMN "social_links"."platform_user_id" IS 'YouTube channelId';

COMMENT ON COLUMN "payouts"."chain" IS '''ethereum''';

COMMENT ON COLUMN "payouts"."address" IS '0x...';

COMMENT ON COLUMN "ledger_events"."platform" IS '''youtube''';

COMMENT ON COLUMN "ledger_events"."platform_user_id" IS 'channelId';

COMMENT ON COLUMN "ledger_events"."user_id" IS 'nullable until claimed';

COMMENT ON COLUMN "ledger_events"."event_type" IS '''DIRECT'' | ''ESCROW'' | ''WITHDRAW''';

COMMENT ON COLUMN "ledger_events"."amount_raw" IS 'Token base units (MNEE decimals)';

COMMENT ON COLUMN "ledger_events"."message" IS 'Optional tipper message, shown to creator';

ALTER TABLE "identities" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "social_links" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "payouts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "ledger_events" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
