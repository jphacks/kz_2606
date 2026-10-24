package usecase

// 参加中セッションの調停。ここがコア。
// 部屋ごとに goroutine 1本を立て、domain.Room への操作を直列化する（actorパターン）。
// メモリ上の軌跡が真実のソースで、PointRepository への書き出しは5秒ごとの後追い。
// 仕様: DATA_MODEL.md 8章
//
// TODO: SessionService
// TODO: Join(code, member) (*domain.Room, error)  ← 戻り値が snapshot の中身（Clone）
// TODO: StartRoute / AppendPoints / StopRoute / UpdateMember / Leave
// TODO: ルームごとの actor ループ / flush（5秒ごとにまとめて保存）
//
// ⚠️ Leave の時点で開いている route を自動で閉じること。
//    忘れると ended_at が永久にNULLの route が溜まり「記録中」が消えなくなる。
