package ws

// ワイヤ形式と domain の変換。Point は [lat, lng, ts, acc] のタプルにする。
// ワイヤ形式を変えるときはこのファイルだけ直せばよい。
// 仕様: DATA_MODEL.md 3章・6章
//
// ★ ここだけは go test を書く。タプル変換のバグはフロントから原因が追いにくい。
//
// TODO: inbound（start / points / stop / member）
// TODO: snapshot / route / points / route_end / member / leave
// TODO: closed / reopened / banned / unbanned / route_deleted / error
