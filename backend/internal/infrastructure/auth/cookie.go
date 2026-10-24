package auth

// usecase.SessionCodec の実装。userID と失効時刻を HMAC-SHA256 で署名する。
// セッションテーブルは作らない（サーバー再起動でログアウトしないのが開発中に効く）。
// 鍵は SESSION_SECRET。有効期限7日。
//
// TODO: SessionCodec / Encode / Decode
