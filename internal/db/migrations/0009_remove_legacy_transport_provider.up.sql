-- 0009_remove_legacy_transport_provider — drop retired transport service row.
--
-- connexto-proxy-cli-llms was a mistaken provider id for the shared Connexto Proxy
-- HTTP service. Logical providers (cursor, and future ids) point at that service
-- via the runtime registry; they are configured under providers.id = 'cursor'.

DELETE FROM providers
WHERE id = 'connexto-proxy-cli-llms';
