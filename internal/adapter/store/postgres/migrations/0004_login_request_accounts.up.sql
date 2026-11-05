-- SPDX-License-Identifier: AGPL-3.0-only
-- The account of each login request (#304): domain.Email.Account, the
-- address in lower case, as api_keys.account, so that the erasure of an
-- account deletes its requests by an index, without reading the others'
-- addresses and without SQL folding case itself. Rows stored before this
-- migration keep no account; the purge deletes them at most
-- domain.LoginRequestRetention after their link ends.
ALTER TABLE login_requests
    ADD COLUMN account text CHECK (char_length(account) BETWEEN 3 AND 254);

CREATE INDEX login_requests_account ON login_requests (account);
