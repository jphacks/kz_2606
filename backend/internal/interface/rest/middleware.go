package rest

// Cookie を検証して context にユーザーを載せる。
// WebSocket のハンドシェイクにも Cookie は乗るので、ws 側からも同じものを使う。
//
// TODO: withUser / userFrom(ctx) / requireUser
