package usecase

// 出力ポート（依存性逆転の境界）。実装は infrastructure と interface。
// usecase は domain 以外の具体型を知らない。
//
// TODO: UserRepository   — users
// TODO: RoomRepository   — rooms / members
// TODO: RouteRepository  — routes（記録開始で作成、終了で ended_at）
// TODO: PointRepository  — points のバルク保存（CopyFrom）、部屋単位の読み出し
// TODO: PasswordHasher   — 実装: infrastructure/auth/bcrypt.go
// TODO: SessionCodec     — 実装: infrastructure/auth/cookie.go
// TODO: Notifier         — 実装: interface/ws/registry.go
//                          NotifyPoints / NotifyRoute / NotifyRouteEnd / NotifyMember
//                          NotifyLeave / NotifyClosed / NotifyBanned / NotifyRouteDeleted
