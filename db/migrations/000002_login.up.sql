CREATE TABLE login_nonces (
  address varchar PRIMARY KEY,
  nonce varchar NOT NULL,
  expires_at timestamp NOT NULL,
  created_at timestamp NOT NULL DEFAULT NOW()
);
