-- SPDX-License-Identifier: AGPL-3.0-only
DROP INDEX login_requests_account;
ALTER TABLE login_requests DROP COLUMN account;
