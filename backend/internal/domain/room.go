package domain

// Room 集約。コード・メンバー・軌跡を持ち、状態遷移のルールを持つ。
// **並行制御はしない**（usecase/session.go が goroutine 1本で直列化する）。
//
// TODO: Room{Code, Members, Tracks} / Join / Leave / AddPoints / UpdateMember
