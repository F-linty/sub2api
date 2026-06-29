-- CockroachDB variant of 108a. Flatten the DO-block length guard to a plain widen;
-- CRDB rejects ALTER TABLE inside function/DO definitions.
ALTER TABLE auth_identity_migration_reports ALTER COLUMN report_type TYPE VARCHAR(80);
