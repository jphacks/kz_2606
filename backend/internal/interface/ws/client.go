package ws

// WebSocket 1接続分。read pump と write pump。
// ping は30秒ごと（Cloudflare は無通信100秒で切る）。送信バッファが溢れたら切断する。
//
// TODO: client{conn, out, member} / readPump / writePump
