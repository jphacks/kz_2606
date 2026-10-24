.RECIPEPREFIX = >

# .env があれば読み込む（DATABASE_URL など）
ifneq (,$(wildcard .env))
include .env
export
endif
.PHONY: help db db-down tidy server web build run clean

help:
> @grep -E '^[a-z-]+:' Makefile | sed 's/:.*//' | sed 's/^/make /'

db:            ## Postgres を起動
> docker compose up -d db

db-down:
> docker compose down

tidy:          ## go.sum を生成（初回必須）
> cd backend && go mod tidy

server:        ## Go を起動（:8080）
> cd backend && go run ./cmd/server

web:           ## Vite dev を起動（:5173, /api と /ws は :8080 へproxy）
> cd frontend && npm run dev

build:         ## フロントをビルドして単一バイナリを作る
> cd frontend && npm ci && npm run build
> cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../gpsart ./cmd/server

run: build
> ./gpsart

clean:
> rm -f gpsart
> rm -rf backend/cmd/server/dist/* frontend/dist
> touch backend/cmd/server/dist/.gitkeep
