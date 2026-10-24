package memory

// DB 無しで動かすための実装（デモ保険 / 初期開発用）。
// DATABASE_URL が空のときに persistence の代わりに差し替える。
//
// TODO: RoomRepository / PointRepository（どちらも no-op か map 保持）
