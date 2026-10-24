package persistence

// pgxpool の接続と、起動時の schema.sql 実行。
// DATABASE_URL が空なら接続しない（infrastructure/memory にフォールバックする）。
//
// TODO: //go:embed schema.sql
// TODO: Open(ctx, url) (*pgxpool.Pool, error)
