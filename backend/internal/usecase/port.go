package usecase

// 出力ポート（依存性逆転の境界）。実装は infrastructure と interface。
// usecase は domain 以外の具体型を知らない。
//
// TODO: RoomRepository  — 部屋とメンバーの永続化、復元
// TODO: PointRepository — 点のバルク保存、部屋単位の読み出し
// TODO: Notifier        — 部屋のメンバーへメッセージを配る（実装: interface/ws/registry.go）
