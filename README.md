# まちえ（machie）

**一人でも、みんなでも。歩いた軌跡をアートに。**

[![IMAGE ALT TEXT HERE](https://jphacks.com/wp-content/uploads/2026/07/IMG_5316.png)](https://www.youtube.com/watch?v=piaNsc6ilBI)

## 製品概要

### 背景(製品開発のきっかけ、課題等）

<!-- TODO: チームの実体験・きっかけを足してください。以下はプロダクトの課題設定です -->

GPSの軌跡で地図に絵を描く「GPSアート」は以前からある遊びですが、**基本的に一人の作業**でした。走ったログを後から見て楽しむもので、誰かと一緒に描くには「どこを担当するか」を口頭で決めて、終わってからログを重ねて確認するしかありません。

一方で「歩く」こと自体は続けにくい習慣です。歩数計のグラフが増えても、それだけでは外に出る理由になりません。

**歩くことに「共同作業」という目的を与えられないか** という発想から始めました。同じ地図の上に複数人の軌跡がその場で描かれていけば、歩くことが協力プレイになり、できあがった絵が記録として残ります。

### 製品説明（具体的な製品の説明）

「まちえ」は、**みんなで歩いて1枚の地図に絵を描く Webアプリ**です。

1. 誰かが部屋を作ると、6文字の部屋コードと招待QRが出ます
2. 参加者はQRをスマホの**標準カメラで読み取るだけ**で参加できます（アプリのインストールは不要）
3. 記録を始めると、歩いた軌跡が**メンバーごとの色の線**で、全員の地図に同時に描かれていきます
4. 描き終わったら部屋を「締める」と、**書き込みは止まり、絵はそのまま見られる状態**で残ります

一人で描きたいときのために、**アカウントを作ると自動で自分専用の個人ルーム**が用意されます。誰も入ってこないので、一人のGPSアートはそこで完結します。

### 特長

#### 1. 複数人の軌跡が、その場で1枚の地図に集まる

WebSocket で全員に配信するので、**他のメンバーの線が歩いている最中に伸びていきます**。「ここは私が担当」が地図を見ながら決まります。接続した瞬間に現在の絵がまるごと届く（snapshot）ので、後から参加しても、回線が切れて入り直しても、同じ絵から再開できます。

#### 2. 6文字のコードとQRだけで参加できる

部屋コードは `I` `O` `0` `1` を除いた32文字から6文字。**読み上げても聞き間違えにくく**、QRにすればカメラを向けるだけです。読み取りはOS標準のカメラでよく、アプリ内にスキャナを持ちません。URLは `?room=KZ26AB` の形なので、LINEで送っても機能します。

#### 3. 締めたあとも「見る」ことはできる

部屋を締めると書き込みは禁止になりますが、**閲覧は自由なまま**です。描き終わった絵のQRをそのまま人に見せられるので、作品として渡せます。誤って締めても再開できます。

### 解決出来ること

- **歩く理由ができる** — 歩数ではなく「絵の担当区間」という目的で外に出られます
- **離れていても一緒に描ける** — 同じ場所に集まらなくても、同じ地図の上では共同作業になります
- **成果が形で残る** — 歩いた結果が1枚の絵として残り、距離・時間とあわせて振り返れます
- **参加のハードルが低い** — インストール不要、QRを読むだけ。歩くこと以外の準備が要りません

### 今後の展望

<!-- TODO: チームでやりたいことを足してください。以下は設計上すぐ繋がるものです -->

- **お題モード** — 「ハートを描く」などの下絵を地図に重ね、担当区間を自動で割り振る
- **作品ギャラリー** — 締めた部屋を一覧で見られるようにして、他のチームの絵を鑑賞できるようにする
- **軌跡の再生** — 点が時刻を持っているので、描かれていく過程をアニメーションで再生できます（追加のデータ構造は不要）
- **歩数・消費カロリーとの連携** — ヘルスケア系のデータと組み合わせて運動記録としての価値を足す

### 注力したこと（こだわり等）

* **GPSの生データをそのまま描かないこと。** 生の測位をそのまま線にすると、立ち止まったところが団子になり、測位の飛びがヒゲになります。「精度30m超は捨てる／直前から5m未満は捨てる／速度14m/s超は捨てる／ただし30秒以上空いたあとの最初の点は捨てない」という採否ルールを決めて、クライアント側で確定させました
* **回線が切れる前提で考えること。** 歩きながらの記録では電波は切れます。WebSocketは30秒ごとのpingで維持し、送信バッファが溢れたら切る、接続時には現在の絵をまるごと送り直す、という方針にしました。距離の計算も「30秒を超えて空いた区間は数えない」と決めて、サーバーとクライアントで**同じ境界**に揃えています
* **仕様書を単一の正にすること。** DBスキーマ・REST・WebSocketのメッセージ形式・権限・定数を1つのドキュメント（[docs/DATA_MODEL.md](docs/DATA_MODEL.md)）に集め、実装で決めたことは必ずそこへ反映する運用にしました。連動する定数には「片方だけ変えるとズレる」と相手を明記しています

## 開発技術

### 活用した技術

#### API・データ

* **Geolocation API**（`watchPosition`）— 位置情報の取得
* **WebSocket** — 軌跡のリアルタイム共有
* **OpenStreetMap** タイル — 地図表示
* **PostgreSQL 16** — 部屋・メンバー・軌跡・点列の保存（PostGISは使わず素のPostgres）

#### フレームワーク・ライブラリ・モジュール

**フロントエンド**

* React 19 / TypeScript 5.7 / Vite 6
* Tailwind CSS v4
* Leaflet 1.9 + react-leaflet 5（軌跡の描画は Leaflet のレイヤを直接操作）
* qrcode（招待QRの生成）
* lucide-react（アイコン）

**バックエンド**

* Go 1.26（HTTPは標準ライブラリの `net/http` のみ。Webフレームワークなし）
* gorilla/websocket
* jackc/pgx v5（`pgxpool`）
* golang.org/x/crypto（bcrypt）
* 構成: `domain` / `usecase` / `interface` / `infrastructure` のレイヤ分割

#### デバイス

<!-- TODO: 実際に動作確認した端末を書いてください -->

* スマートフォン（GPS・ブラウザのGeolocation APIが使えるもの）
* PCブラウザ（閲覧・部屋の管理）

### 独自技術

#### ハッカソンで開発した独自機能・技術

* **部屋の中で「編集可」な軌跡は常に1本という不変条件** — 誰かが記録している間は他の人が記録を始められないようにし、DBの部分ユニーク索引（`routes_one_recording_per_room`）で保証しています。同時にリクエストが来ても2本目が作られません
  → [backend/internal/infrastructure/persistence/schema.sql](backend/internal/infrastructure/persistence/schema.sql)

* **サーバーとクライアントで同じ距離計算** — haversineでの累積距離を両側に実装し、「30秒を超えて空いた区間は加算しない」という境界まで揃えました。数字が食い違わないよう、定数表に連動相手を明記しています
  → [backend/internal/domain/distance.go](backend/internal/domain/distance.go) / [frontend/src/lib/geo.ts](frontend/src/lib/geo.ts)

* **点列をReactの状態に載せない描画** — 軌跡は毎秒頂点が増えるため、Reactの状態に入れると毎秒アプリ全体が再描画されます。点列を専用のバッファに持たせ、描画側は Leaflet のポリラインを直接伸ばす形にしました
  → [frontend/src/lib/track_buffer.ts](frontend/src/lib/track_buffer.ts) / [frontend/src/ui/map.tsx](frontend/src/ui/map.tsx)

* **アカウント作成と同時に個人ルームを作る** — 一人でもすぐ始められるように、登録・ログイン・セッション確認のいずれでも自分の個人ルームが返るようにしました。1ユーザー1つを部分ユニーク索引で保証しています
  → [backend/internal/usecase/auth.go](backend/internal/usecase/auth.go)

* **認証まわりの作り込み** — セッションはHMAC-SHA256署名付きのHttpOnly Cookie（期限を署名対象に含める）。ログインは「ユーザーが存在しない」ケースでもハッシュ比較のコストを払って、応答時間からユーザー名を列挙できないようにしています。JSONエンドポイントは `Content-Type: application/json` を必須にして、プリフライトなしの外部サイトからのPOSTを防いでいます
  → [backend/internal/infrastructure/auth/cookie.go](backend/internal/infrastructure/auth/cookie.go) / [backend/internal/usecase/auth.go](backend/internal/usecase/auth.go)

* **切断を前提にしたWebSocket** — 30秒ごとのping（無通信100秒で切られる環境を想定）、受信64KB上限、送信バッファが溢れたら切断。接続直後のsnapshotは必ず差分より先に届くようにして、「土台のない差分が届く」状態が起きない作りにしています
  → [backend/internal/interface/ws/client.go](backend/internal/interface/ws/client.go) / [backend/internal/interface/ws/registry.go](backend/internal/interface/ws/registry.go)

## ドキュメント

* [docs/DATA_MODEL.md](docs/DATA_MODEL.md) — データ構造・REST・WebSocketプロトコル・権限・定数の仕様（単一の正）
* [docs/TECH_PLAN.md](docs/TECH_PLAN.md) — 技術選定の方針と、詰まりやすいポイントの整理
* [docs/BUILD.md](docs/BUILD.md) — ローカルでの起動方法

## ローカルでの起動

```bash
# PostgreSQL を起動
docker compose up -d db

# バックエンド（:8080）
cd backend && go run ./cmd/server

# フロントエンド（:5173。/api と /ws は :8080 にプロキシ）
cd frontend && npm install && npm run dev
```

詳細と環境変数（`DATABASE_URL` / `PORT` / `SESSION_SECRET` / `COOKIE_SECURE`）は [docs/BUILD.md](docs/BUILD.md) を参照してください。
