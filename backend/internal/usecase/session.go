package usecase

// 参加中セッションの調停。ここがコア。
// 部屋ごとに goroutine 1本を立て、Room 集約への操作を直列化する（actorパターン）。
// メモリ上の軌跡が真実のソースで、PointRepository への書き出しは5秒ごとの後追い。
// 仕様: DATA_MODEL.md 7章
//
// TODO: SessionService / Join / Leave / AppendPoints / UpdateMember
// TODO: ルームごとの actor ループ / flush（5秒ごとにまとめて保存）
