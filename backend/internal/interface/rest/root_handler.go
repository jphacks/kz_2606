package rest

import (
	"io"
	"log/slog"
	"net/http"
)

func rootHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := io.WriteString(w, "hello backend"); err != nil {
		slog.Error("write root response failed", "err", err)
	}
}
