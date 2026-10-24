# ビルド・起動

## ローカルで起動

Docker Compose で PostgreSQL を起動してから、リポジトリのルートでバックエンドを実行します。

```powershell
docker compose up -d db
go run backend/cmd/server/main.go
```

既定では `localhost:5432` の開発用 PostgreSQL とポート `8080` を使います。
`DATABASE_URL`、`PORT`、`SESSION_SECRET` を環境変数で設定すれば上書きできます。
`SESSION_SECRET` を省略した場合は一時鍵を生成するため、サーバー再起動でログインセッションが無効になります。
