package ws

// WebSocket のハンドシェイク。
// Cookie から本人を特定し、?room=CODE の member でなければ 403 で切る。
// クライアント申告のIDは一切受け取らない。接続直後に snapshot を送る。
// 仕様: DATA_MODEL.md 6章
//
// TODO: Handler / ServeHTTP / upgrader
