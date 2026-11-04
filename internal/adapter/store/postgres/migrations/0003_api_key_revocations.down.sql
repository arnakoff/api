-- SPDX-License-Identifier: AGPL-3.0-only
DROP TRIGGER api_key_revoked ON api_keys;
DROP FUNCTION api_key_revoked_notify();
