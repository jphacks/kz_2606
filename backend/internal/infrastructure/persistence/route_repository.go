package persistence

// usecase.RouteRepository の Postgres 実装。
// 記録開始で1行 INSERT（同期。points より先に存在させる）、終了で ended_at を UPDATE。
//
// TODO: RouteRepository / Create / End / Delete / FindByRoom
