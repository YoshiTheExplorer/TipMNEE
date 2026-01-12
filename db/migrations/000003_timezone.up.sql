BEGIN;

-- login_nonces is ephemeral (short-lived), so we can safely recreate it.
DROP TABLE IF EXISTS login_nonces;

CREATE TABLE login_nonces (
  address    varchar PRIMARY KEY,
  nonce      text        NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT NOW()
);

COMMIT;