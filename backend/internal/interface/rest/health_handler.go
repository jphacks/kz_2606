package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(struct {
		OK bool `json:"ok"`
	}{OK: true}); err != nil {
		slog.Error("write health response failed", "err", err)
	}
}
