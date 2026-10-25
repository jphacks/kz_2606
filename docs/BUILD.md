# ビルド・起動

## ローカルで起動

Docker Compose で PostgreSQL を起動してから、リポジトリのルートでバックエンドを実行します。

```powershell
docker compose up -d db
cd backend
go run ./cmd/server
```

既定では `localhost:5432` の開発用 PostgreSQL とポート `8080` を使います。
`DATABASE_URL`、`PORT`、`SESSION_SECRET`、`COOKIE_SECURE` を環境変数で設定すれば上書きできます。
`SESSION_SECRET` を省略した場合は一時鍵を生成するため、サーバー再起動でログインセッションが無効になります。
`COOKIE_SECURE` は既定で `true`。`http://` の開発サーバー（スマホ実機を LAN でつなぐ場合など）では
ブラウザが Secure Cookie を捨ててログインできないので、その場合だけ `COOKIE_SECURE=false` を指定します。
