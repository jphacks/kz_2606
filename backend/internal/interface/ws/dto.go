package ws

// ワイヤ形式と domain の変換。Point は [lat, lng, ts, acc] のタプルにする。
// ワイヤ形式を変えるときはこのファイルだけ直せばよい。
// 仕様: DATA_MODEL.md 3章・5章
//
// TODO: pointDTO(Marshal/UnmarshalJSON) / inbound / snapshot / points / member / leave
