# ビルド・起動

## ローカルで起動

Docker Compose で PostgreSQL を起動してから、バックエンドとフロントエンドをそれぞれ起動します。

### PostgreSQL

```powershell
docker compose up -d db
```

### バックエンドサーバー

サーバーは既定で `http://localhost:8080` で起動します。

```powershell
cd backend
go run ./cmd/server
```

### フロントエンド

フロントエンド開発サーバーは `http://localhost:5173` で起動します。
`/api` と `/ws` はバックエンドの `localhost:8080` へプロキシされます。

```powershell
cd frontend
npm install
npm run dev
```

既定では `localhost:5432` の開発用 PostgreSQL とポート `8080` を使います。
`DATABASE_URL`、`PORT`、`SESSION_SECRET`、`COOKIE_SECURE` を環境変数で設定すれば上書きできます。
`SESSION_SECRET` を省略した場合は一時鍵を生成するため、サーバー再起動でログインセッションが無効になります。
`COOKIE_SECURE` は既定で `true`。`http://` の開発サーバー（スマホ実機を LAN でつなぐ場合など）では
ブラウザが Secure Cookie を捨ててログインできないので、その場合だけ `COOKIE_SECURE=false` を指定します。
