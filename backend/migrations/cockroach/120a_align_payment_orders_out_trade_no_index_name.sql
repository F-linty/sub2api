-- CockroachDB variant of 120a. Flatten the DO-block rename guard. IF EXISTS keeps it
-- idempotent and safe if the source index is already named/aligned.
DROP INDEX IF EXISTS paymentorder_out_trade_no;
ALTER INDEX IF EXISTS paymentorder_out_trade_no_unique RENAME TO paymentorder_out_trade_no;
