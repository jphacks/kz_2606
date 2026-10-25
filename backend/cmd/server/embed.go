package main

import (
	"embed"
	"io/fs"
)

// frontend のビルド出力を埋め込む。
// go:embed は親ディレクトリを辿れないため、Vite の outDir をこの階層に向けている
// （frontend/vite.config.ts の outDir = ../backend/cmd/server/dist）。

//go:embed all:dist
var webFiles embed.FS

func staticFS() (fs.FS, error) {
	return fs.Sub(webFiles, "dist")
}
