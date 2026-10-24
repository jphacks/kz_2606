package rest

import "net/http"

// ServeMux の組み立て。CORS とセッションミドルウェアもここ。
// 仕様: DATA_MODEL.md 5章

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", rootHandler)
	return mux
}
