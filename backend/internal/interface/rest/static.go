package rest

// 埋め込み済みフロントの配信。fs.FS を受け取るだけで embed はしない（cmd/server の責務）。
// 未知のパスは index.html を返す（SPAフォールバック）。
//
// TODO: staticHandler(fs.FS)
