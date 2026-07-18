-- 154_account_spark_shadow.sql
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS parent_account_id BIGINT,
    ADD COLUMN IF NOT EXISTS quota_dimension VARCHAR(20) NOT NULL DEFAULT 'global';

-- CockroachDB compatibility:
-- CockroachDB does not support ALTER TABLE statements inside DO/function bodies.
-- PostgreSQL does not support ADD CONSTRAINT IF NOT EXISTS, so use explicit
-- drop-and-add statements to keep the migration retry-safe on both databases.
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS fk_accounts_parent_account_id;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_parent_not_self;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_parent_dimension;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_quota_dimension;

ALTER TABLE accounts ADD CONSTRAINT chk_accounts_quota_dimension
    CHECK (quota_dimension IN ('global','spark'));

ALTER TABLE accounts ADD CONSTRAINT chk_accounts_parent_dimension
    CHECK ((parent_account_id IS NULL AND quota_dimension = 'global')
        OR (parent_account_id IS NOT NULL AND quota_dimension <> 'global'));

ALTER TABLE accounts ADD CONSTRAINT chk_accounts_parent_not_self
    CHECK (parent_account_id IS NULL OR parent_account_id <> id);

ALTER TABLE accounts ADD CONSTRAINT fk_accounts_parent_account_id
    FOREIGN KEY (parent_account_id) REFERENCES accounts(id) ON DELETE RESTRICT;
