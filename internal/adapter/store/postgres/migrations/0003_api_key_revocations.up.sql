-- SPDX-License-Identifier: AGPL-3.0-only
-- A change to an API key, or its deletion, is announced on the channel
-- api_key_revoked with the hex digest of the key, so that every running
-- API forgets it at once (#246). Keys only change when they are revoked;
-- the notification is sent at commit, never for a rolled-back change.
-- search_path is fixed so that no schema can shadow pg_notify or encode.
CREATE FUNCTION api_key_revoked_notify() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $$
BEGIN
    PERFORM pg_notify('api_key_revoked', encode(OLD.digest, 'hex'));
    RETURN NULL;
END
$$;

CREATE TRIGGER api_key_revoked AFTER UPDATE OR DELETE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION api_key_revoked_notify();
