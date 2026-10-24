package rest

// 部屋のハンドラ。
//   POST /api/rooms              作成。**レスポンスに code を含める**（作成者は即QRを出す）
//   GET  /api/rooms/{code}       存在確認のみ。軌跡は返さない（それは snapshot の仕事）
//   POST /api/rooms/{code}/join  members 行を作る。冪等（参加済みなら既存を返す）
// 仕様: DATA_MODEL.md 5章
//
// TODO: roomHandler / create / get / join
