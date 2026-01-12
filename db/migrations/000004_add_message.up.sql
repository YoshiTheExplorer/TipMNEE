ALTER TABLE login_nonces
ADD COLUMN IF NOT EXISTS message text;

UPDATE login_nonces
SET message = ''
WHERE message IS NULL;

ALTER TABLE login_nonces
ALTER COLUMN message SET NOT NULL;
