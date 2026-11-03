-- SPDX-License-Identifier: AGPL-3.0-only
DROP TABLE login_requests;
DROP INDEX api_keys_active_account;
ALTER TABLE api_keys DROP COLUMN account, DROP COLUMN email;
