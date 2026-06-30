-- CockroachDB variant of 116. The PostgreSQL original's legacy-data scan is a no-op on
-- fresh installs (no user_external_identities table). The only effects that matter for a
-- fresh schema are the three jsonb-object CHECK constraints, flattened here from the
-- DO-block guards. The auth_identity* tables are empty on a fresh install, so validation
-- passes immediately.
ALTER TABLE auth_identities DROP CONSTRAINT IF EXISTS auth_identities_metadata_is_object_check;
ALTER TABLE auth_identities
    ADD CONSTRAINT auth_identities_metadata_is_object_check
    CHECK (jsonb_typeof(metadata) = 'object');

ALTER TABLE auth_identity_channels DROP CONSTRAINT IF EXISTS auth_identity_channels_metadata_is_object_check;
ALTER TABLE auth_identity_channels
    ADD CONSTRAINT auth_identity_channels_metadata_is_object_check
    CHECK (jsonb_typeof(metadata) = 'object');

ALTER TABLE auth_identity_migration_reports DROP CONSTRAINT IF EXISTS auth_identity_migration_reports_details_is_object_check;
ALTER TABLE auth_identity_migration_reports
    ADD CONSTRAINT auth_identity_migration_reports_details_is_object_check
    CHECK (jsonb_typeof(details) = 'object');
