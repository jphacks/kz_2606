// WebSocket クライアント。指数バックオフで自動再接続する。
// 切断中の点は outbox に積んで、再接続時に送り直す。
//
// TODO: RoomSocket{connect, sendPoints, setMember, close}

export {};
