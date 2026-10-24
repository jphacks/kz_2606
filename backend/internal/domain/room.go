package domain

// Room 集約。コード・メンバー・route・軌跡を持ち、状態遷移のルールを持つ。
// **並行制御はしない**（usecase/session.go が goroutine 1本で直列化する）。
// 仕様: DATA_MODEL.md 7章・8章
//
// TODO: Room{ID, Code, Title, OwnerUserID, ClosedAt, Members, Routes, Active}
// TODO: Join / Leave / UpdateMember / Ban / Unban / Close / Reopen
// TODO: StartRoute / AddPoints / EndRoute / DeleteRoute
// TODO: Clone()  ← snapshot 用。実体を外に出さないために必須
// TODO: CanWrite() / IsOwner()
