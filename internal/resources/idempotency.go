package resources

// Idempotency retention (ADR-023): resource_admissions rows are keyed by the
// stable internal admission identifier and are never updated after insert.
// Retries reuse the same identifier until the row is visible, so counters
// cannot double-charge. Operators may delete rows older than the previous UTC
// day during maintenance; that window exceeds any in-process transaction retry.
