package main

// frontend のビルド出力を埋め込む。
// go:embed は親ディレクトリを辿れないため、Vite の outDir をこの階層に向けている
// （frontend/vite.config.ts の outDir = ../backend/cmd/server/dist）。
//
// TODO: //go:embed all:dist
// TODO: staticFS() fs.FS
