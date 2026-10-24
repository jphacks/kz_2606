package persistence

// usecase.PointRepository の Postgres 実装。
// 保存は pgx の CopyFrom で一括。1点ずつ INSERT しない。
//
// TODO: PointRepository / Save / FindByRoom
