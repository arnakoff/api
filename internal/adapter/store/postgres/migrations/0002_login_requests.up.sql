-- SPDX-License-Identifier: AGPL-3.0-only
-- Free keys by login link (#59, ADR 0006): the address of the account a key
-- is issued to, kept with the key and deleted with it, and the login
-- requests, kept with the digest of their token, never the token.
-- account is domain.Email.Account, the address in lower case: the identity
-- the API compares, so that SQL never folds case itself.
ALTER TABLE api_keys
    ADD COLUMN email   text CHECK (char_length(email) BETWEEN 3 AND 254),
    ADD COLUMN account text CHECK (char_length(account) BETWEEN 3 AND 254),
    ADD CHECK ((email IS NULL) = (account IS NULL));

-- One active key per account.
CREATE UNIQUE INDEX api_keys_active_account ON api_keys (account)
    WHERE revoked_at IS NULL AND account IS NOT NULL;

CREATE TABLE login_requests (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_digest bytea       NOT NULL UNIQUE CHECK (octet_length(token_digest) = 32),
    email        text        NOT NULL CHECK (char_length(email) BETWEEN 3 AND 254),
    created_at   timestamptz NOT NULL,
    expires_at   timestamptz NOT NULL CHECK (expires_at > created_at),
    used_at      timestamptz CHECK (used_at >= created_at AND used_at < expires_at)
);

-- The purge deletes by the end of the link: used, or else expired.
CREATE INDEX login_requests_ended ON login_requests (COALESCE(used_at, expires_at));
