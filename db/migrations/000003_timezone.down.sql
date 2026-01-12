BEGIN;

DROP TABLE IF EXISTS login_nonces;

CREATE TABLE login_nonces (
  address    varchar PRIMARY KEY,
  nonce      text      NOT NULL,
  expires_at timestamp NOT NULL,
  created_at timestamp NOT NULL DEFAULT NOW()
);

COMMIT;