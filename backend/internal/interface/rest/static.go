package rest

import (
	"bytes"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
)

// StaticHandler は埋め込み済みフロントの静的ファイルを配信し、
// 未知の画面パスでは index.html を返す。embed 自体は cmd/server の責務。
func StaticHandler(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath := path.Clean("/" + r.URL.Path)
		if requestPath == "/api" || strings.HasPrefix(requestPath, "/api/") ||
			requestPath == "/ws" || strings.HasPrefix(requestPath, "/ws/") {
			http.NotFound(w, r)
			return
		}

		filePath := strings.TrimPrefix(requestPath, "/")
		if filePath != "" {
			if _, err := fs.Stat(files, filePath); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			} else if !errors.Is(err, fs.ErrNotExist) {
				slog.Error("inspect embedded static file failed", "path", filePath, "err", err)
				http.Error(w, "サーバーエラーが発生しました", http.StatusInternalServerError)
				return
			}
			if path.Ext(filePath) != "" {
				http.NotFound(w, r)
				return
			}
		}

		index, err := fs.ReadFile(files, "index.html")
		if err != nil {
			slog.Error("read embedded SPA entrypoint failed", "err", err)
			http.Error(w, "サーバーエラーが発生しました", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
	})
}
