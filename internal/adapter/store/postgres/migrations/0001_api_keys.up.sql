-- SPDX-License-Identifier: AGPL-3.0-only
-- API keys (#57, ADR 0004): the HMAC-SHA256 digest of each key, never the key.
CREATE TABLE api_keys (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    digest      bytea       NOT NULL UNIQUE CHECK (octet_length(digest) = 32),
    environment text        NOT NULL CHECK (environment IN ('live', 'test')),
    label       text        NOT NULL CHECK (char_length(label) BETWEEN 1 AND 100),
    hint        text        NOT NULL CHECK (char_length(hint) = 4),
    created_at  timestamptz NOT NULL,
    revoked_at  timestamptz CHECK (revoked_at >= created_at)
);
