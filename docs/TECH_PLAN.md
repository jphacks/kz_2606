# 技術方針メモ（共同参加型 GPSアート Webアプリ）

企画案に対する技術スタックの判定と、ハッカソンで実際に詰まるポイントの整理。

## 0. 結論サマリ

| 項目 | 判定 | 補足 |
|---|---|---|
| React + TypeScript + Vite | ◎ | そのままでOK |
| Leaflet.js | ◎ | 短期開発なら正解。MapLibreは綺麗だが学習コストが要る |
| Go + WebSocket | ◎ | 問題なし |
| **PostGIS** | **✗ 不要** | 素のPostgresで十分 |
| **Cloudflare Tunnel** | **△** | 開発中は◎、デモ当日は危険 |
| **バックグラウンドGPS** | **✗ 原理的に無理** | 最大リスク。UXで回避する |

下2つが企画の成否に直結する。

**ライブラリ・デプロイ先まで確定させた一覧は [7. 確定技術スタック](#7-確定技術スタック) を参照。**
**スキーマ・WSプロトコル・定数の仕様は [DATA_MODEL.md](./DATA_MODEL.md) にまとめてある。**

---

## 1. バックグラウンドGPSは諦めて、UXで回避する

Webには Background Geolocation API が**存在しない**。`watchPosition` は画面ロック・タブ切り替えで停止する（iOS Safariは確実に、Android Chromeも画面オフで停止）。PWA化しても解決しない。
ここを後回しにすると当日破綻するので、最初に方針を固める。

### 現実的な設計

- **Screen Wake Lock API** で画面を消させない（iOS 16.4+ / Android Chrome 対応済み）

  ```ts
  const lock = await navigator.wakeLock.request('screen'); // 記録開始時
  // visibilitychange での復帰時に再取得が必要
  ```

- UIを「**スマホを手に持って見ながら歩く**」前提にする。
  GPSアートは軌跡が伸びるのを眺めるのが楽しいので、これは不自然ではなく、むしろ機能として成立する。
- それでも落ちるので **復帰時のギャップ検出** を入れる。
  最後の点から30秒以上空いたら、その区間を点線で描く or 直線補間する。
  「取れていない」状態を事故ではなく仕様として見せる。
- 無音オーディオをループ再生してバックグラウンド実行を維持する裏技は、iOSでは位置情報まで生き残らないことが多い。保険として試す程度に留める。

---

## 2. DBはPostGIS不要、構成を1段削る

やることは `(team_id, user_id, lat, lng, accuracy, recorded_at)` を時系列でINSERTして順番に読み出すだけ。空間インデックスも空間関数も使わない。
**素のPostgres（docker compose 一発）** で十分。

```sql
CREATE TABLE points (
  id          BIGSERIAL PRIMARY KEY,
  room_id     TEXT NOT NULL,
  member_id   TEXT NOT NULL,
  lat         DOUBLE PRECISION NOT NULL,
  lng         DOUBLE PRECISION NOT NULL,
  accuracy    REAL,
  recorded_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX ON points (room_id, member_id, recorded_at);
```

「塗りつぶし面積スコア」のような計算をやりたくなっても、フロントで turf.js を使うほうが速く作れる。

### 書き込み方の注意

1点ごとにINSERTしない。クライアントで3秒ごとにバッチ化してWebSocketで送り、サーバーは
Postgresのトランザクション内で `CopyFrom` によって保存する。**保存成功後に他の参加者へ
ブロードキャスト**するため、通知された点はsnapshotからも復元できる。

---

## 3. トンネルは「開発用」。デモ当日は本番デプロイしておく

Cloudflare Tunnel自体は良い選択だが、罠が2つある。

- 無料の `trycloudflare.com` クイックトンネルは**再起動ごとにURLが変わる**。
  固定URLにするには独自ドメインをCloudflareに載せる必要がある（ドメイン代 年1000円程度）。
  固定URLだけ欲しいなら、静的ドメインが1個もらえる **ngrok無料枠** のほうが楽。
- より怖いのが**会場Wi-Fi**。
  「自分のノートPC + トンネル + 審査員のスマホ数台」はネットワークが混むと普通に落ちる。

### おすすめの着地

- **開発中**: Cloudflare Tunnel（スマホの位置情報取得にHTTPSが必須なので妥当）
  - PC上のlocalhostは secure context 扱いなので、PCだけで開発する間はトンネル不要
- **当日**:
  - フロント → **Cloudflare Pages**（無料）
  - Goバックエンド → **Koyeb** or **Fly.io**（WebSocket対応）
  - Render無料枠は15分で寝て復帰に1分近くかかるのでデモには不向き
- **初日の夕方までにデプロイを通しておく**。最終日に初デプロイは事故パターン。

> 余談: Cloudflareを使うなら **Durable Objects**（部屋ごとに1インスタンス、WebSocket + ストレージ内蔵）がこの用途に完璧にハマる。ただしTypeScript。Goで書きたいなら無視してよい。

---

## 4. 地味だが成果物の見栄えを決めるところ

GPSアートは**生データのままだと汚い**。ここが一番効く。

- `accuracy > 30m` の点は捨てる
- 前の点から **5m未満は捨てる**（立ち止まるとジッターで団子になる）
- 速度50km/h超の飛びは捨てる（基地局測位の誤爆）
- 描画時は Douglas-Peucker で間引き + Leafletの `smoothFactor`
- WS切断中は IndexedDB にバッファして再接続時にリプレイ（モバイル回線は普通に切れる）

---

## 5. 審査でウケる足し算（実装が安い順）

1. **リプレイのタイムラプス** — タイムスタンプ付きの点列があるので、スライダーで時間を進めるだけ。実装数時間、デモのインパクトは最大。**優先度最高**。
2. **お題レイヤー** — 目標の文字/図形をゴーストで地図に重ね、達成率%を出す。「協力して描く」が一目で伝わる。
3. **塗りつぶしモードはグリッドで** — ポリゴン演算は地獄。固定グリッドのセル塗りにすれば一瞬で、スコア化もできる。
4. **完成図の画像エクスポート** — SNS導線。

---

## 6. 進め方

- **認証は作らない**。名前 + 色を localStorage、部屋は6桁コード + QRコード。
- 最初にやるのは **WebSocketのメッセージスキーマを決めること**。これが決まればフロントとバックが並行で動ける。

```ts
// Client → Server
{ t: 'join',   room: 'ABC123', member: { id, name, color } }
{ t: 'points', pts: [[lat, lng, ts, acc], ...] }

// Server → Client
{ t: 'snapshot', members: [...], tracks: { memberId: [...] } } // 参加時に既存軌跡をまとめて
{ t: 'points',   member: 'xxx', pts: [...] }                   // 他人の増分
```

- **モックGPSモードを最初に作る**。あらかじめ用意したパスを時間で再生して `watchPosition` の代わりに流すだけ。
  これがないと「1行直すたびに外を歩く」ことになる。

### 着手順チェックリスト

- [ ] WSメッセージスキーマを確定
- [ ] モックGPSモードを実装
- [ ] Goのルーム単位WSハブ（インメモリ）+ ブロードキャスト
- [ ] Leafletで自分の軌跡を描画（色分け・localStorage）
- [ ] 他メンバーの軌跡を重ねて描画
- [ ] 間引き・フィルタ処理（5m / accuracy 30m / 速度外れ値）
- [ ] Wake Lock + ギャップ検出
- [ ] Postgres永続化（バッチinsert）+ snapshot復元
- [ ] **デプロイを通す（初日夕方まで）**
- [ ] リプレイのタイムラプス
- [ ] お題レイヤー / 画像エクスポート（余力があれば）

---

## 7. 確定技術スタック

迷ったらこれ、という確定版。

### 7.1 フロントエンド

| 役割 | 採用 | 理由 |
|---|---|---|
| ビルド | Vite + React 19 + TypeScript | 確定 |
| 地図 | **Leaflet 1.9 を生で使う** | react-leaflet は使わない（7.2参照） |
| タイル | **CARTO dark_matter** | 暗い地図＋蛍光色の線が一番映える |
| 状態管理 | zustand | 3KB。Redux不要 |
| スタイル | Tailwind CSS v4 | Viteプラグイン1行。モバイルUIを速く作れる |
| 間引き | simplify-js | Douglas-Peucker、2KB |
| 永続バッファ | idb-keyval | IndexedDBラッパ、600B |
| QR | qrcode | 部屋コードの共有用。読み取りはOS標準カメラでOK |

```bash
npm create vite@latest web -- --template react-ts
npm i leaflet zustand simplify-js idb-keyval qrcode
npm i -D @types/leaflet tailwindcss @tailwindcss/vite
```

### 7.2 react-leaflet を使わない理由

毎秒ポリラインの頂点が増える構造なので、再レンダリングでpropsを流し込む形はすぐ重くなる。
**地図インスタンスを1個 ref で持って、`Map<memberId, L.Polyline>` を命令的に `addLatLng()` する**ほうが速いし短い。
Reactは地図の「外側」のUI（ボタン・色選択・メンバー一覧）だけ担当させる。

```ts
// useTracks.ts のイメージ
const lines = useRef(new Map<string, L.Polyline>());

function append(memberId: string, pts: L.LatLng[], color: string) {
  let line = lines.current.get(memberId);
  if (!line) {
    line = L.polyline([], { color, weight: 5, smoothFactor: 1.5 }).addTo(map);
    lines.current.set(memberId, line);
  }
  pts.forEach(p => line!.addLatLng(p));
}
```

### 7.3 バックエンド（Go）

| 役割 | 採用 | 理由 |
|---|---|---|
| Go | 1.23+ | |
| ルーター | **標準 `net/http` の ServeMux** | 1.22でパスパターン対応済み。chi等不要 |
| WebSocket | **gorilla/websocket** | hubパターンの実例が圧倒的に多い |
| DBドライバ | **pgx/v5（pgxpool）** | `database/sql` を経由しない |
| バルクINSERT | **pgx の `CopyFrom`** | 点列の一括投入にドンピシャ |
| マイグレーション | **schema.sql を起動時に実行するだけ** | golang-migrateは過剰 |
| ORM | **使わない** | クエリ5本。sqlcもcodegen分の手間が勝つ |
| ログ | 標準 `log/slog` | |
| 設定 | `os.Getenv` | viper不要 |
| ID生成 | `crypto/rand`（部屋=6桁Base32、メンバー=hex16） | uuidライブラリ不要 |

```bash
go mod init github.com/<you>/gpsart
go get github.com/gorilla/websocket github.com/jackc/pgx/v5
```

### 7.4 記録データの永続化と接続管理

現在の実装では **Postgresを永続データと記録状態の真実のソース**にする。ルーム単位の
インメモリHub/actorは設けず、記録操作はDBトランザクションで直列化する。

```
Postgres
 ├ rooms / members / routes / points   ← 永続データとsnapshotの読み出し元
 └ routes_one_recording_per_room       ← 部屋内のactive routeを1本に制限

Registry (map[roomCode]set[*client])
 └ WebSocket接続のみ管理。複数タブとオンライン状態を扱う
```

- 接続時の `snapshot` はPostgresから取得する
- `start` / `points` / `stop` はルーム行をロックする。部屋全体で同時に記録できるrouteは1本
- `points` は1メッセージ最大500点を `CopyFrom` で保存し、成功後に送信者以外へブロードキャスト
- 最後の接続が切れた時は、記録中routeを終了するDB処理を非同期で行う

### 7.5 リポジトリ構成とデプロイ

**フロントをGoバイナリに `go:embed` して1デプロイにする**のを強く推奨。
CORS設定もクロスオリジンWSSも消えて、公開URLが1本になる（QRコードも1個で済む）。

```
kz_2606/
├── frontend/                       # Vite。build 出力は ../backend/cmd/server/dist
├── backend/                        # Go。独立モジュール（go.mod はこの中）
│   ├── cmd/server/                 # エントリポイント。配線と go:embed だけ
│   │   └── dist/                   # ← frontend のビルド出力
│   └── internal/
│       ├── domain/                 # エンティティと不変条件。外部依存なし
│       ├── usecase/                # アプリケーションロジック + 出力ポート
│       ├── interface/              # 外界との変換（rest / ws）
│       └── infrastructure/         # 技術的詳細（postgres / memory / config）
├── docker-compose.yml              # postgres のみ
└── Dockerfile                      # multi-stage: node build → go build → distroless
```

**依存の向きは `interface` → `usecase` → `domain` の一方向のみ。**
`infrastructure` は `usecase/port.go` のインターフェースを実装する側なので、
`usecase` から `infrastructure` を import しない（依存性逆転）。配線は `cmd/server` だけが行う。

`go:embed` は親ディレクトリを辿れないので、Vite の `outDir` を `backend/cmd/server/dist` に向けている。

| 層 | 開発時 | 本番 |
|---|---|---|
| フロント | `vite dev`（`/api`, `/ws` をproxy） | Goバイナリに埋め込み |
| Go | `go run ./server` | **Koyeb**（無料枠・WS可）or Fly.io |
| Postgres | docker compose | **Neon 無料枠** |
| 公開 | Cloudflare Tunnel | Koyebのドメインそのまま |

> Postgresをアプリと同じホストに置かないこと。Neonの無料枠（マネージド）にすれば、
> 本番とローカルの切り替えが `DATABASE_URL` 1個で済む。

### 7.6 API面（WSとRESTの線引き）

全部WSに寄せず、部屋の作成・参照はRESTに出すと curl でデバッグできて楽。

```
POST /api/rooms            → { code: "ABC123" }
GET  /api/rooms/{code}     → { code, members[], createdAt }
GET  /ws?room=ABC123       → WebSocket（join / points / snapshot）
GET  /*                    → embed した SPA
```

### 7.7 環境変数

```
DATABASE_URL=postgres://...
PORT=8080
ALLOW_ORIGIN=*          # 開発時のみ
```

---

## 8. ドメイン・公開構成

ドメインを取るなら、開発中も本番も固定URLになるのでトンネルの弱点が消える。

### 8.1 どこで買うか → Cloudflare Registrar

| 理由 | 内容 |
|---|---|
| 原価販売 | `.com` 約$10.4/年、`.xyz` 約$3/年、`.dev` 約$12/年 |
| 更新料が上がらない | 「1年目1円 → 更新3000円」のような罠がない |
| WHOIS代行が無料 | 個人情報が出ない |
| **DNS反映が即時** | 最初からCloudflare上にあるのでネームサーバー移管の待ちが無い |

> ⚠️ 他社で買うとICANNルールで**60日間は移管できない**。
> ハッカソン期間中にCloudflareへ移せないので、最初からCloudflareで買う。

TLDは `.com` が無難。**`.dev` / `.app` は HSTS preload 強制でHTTPSしか喋れない**ため、
位置情報取得（secure context必須）の観点ではむしろ都合がよい。安く済ませるなら `.xyz`。

### 8.2 サブドメインの切り方

1ドメインでチーム全員ぶんカバーできる。

```
gpsart.example.com        → 本番（Koyeb / Fly.io を CNAME）
taro.dev.example.com      → 太郎のlocalhost:8080（Named Tunnel）
hanako.dev.example.com    → 花子のlocalhost:8080
```

各自が自分の Named Tunnel を持てば、**お互いのローカル環境をスマホ実機で常時確認できる**。

### 8.3 Named Tunnel の立て方

クイックトンネル（`trycloudflare.com`）はURLが毎回変わるので、固定したいならこちら。

```bash
cloudflared tunnel login
cloudflared tunnel create gpsart-taro
cloudflared tunnel route dns gpsart-taro taro.dev.example.com
cloudflared tunnel run gpsart-taro
```

設定ファイルの雛形は [`scripts/cloudflared-config.example.yml`](./scripts/cloudflared-config.example.yml)。

### 8.4 必ず踏む罠

#### Cloudflare経由のWebSocketは無通信100秒で切れる

無料プランでもWSは通るが、**アイドル100秒でCloudflareが切断する**。
歩いている間は点が流れるので平時は平気だが、休憩中に切れる。
ping/pongハートビートが必須（`backend/internal/interface/ws/client.go` の `pingPeriod = 30s`）。

クライアント側も指数バックオフ再接続 + IndexedDBバッファの再送をセットで入れる
（`frontend/src/lib/ws.ts`, `frontend/src/lib/outbox.ts`）。

#### SSL/TLSモードは「Full (strict)」

本番をKoyeb/FlyにCNAMEする場合、デフォルトの **Flexible だとリダイレクトループ**する。
Cloudflareダッシュボードで `Full (strict)` に変更する。

#### dev用サブドメインは保護する

`*.dev.example.com` がノートPCに直結するので、**Cloudflare Access**（無料50ユーザーまで）で
メール認証をかけておく。本番の `gpsart.example.com` にはかけない。

### 8.5 おまけ

- **Cloudflare Email Routing** が無料。`team@example.com` → 各自のGmail転送が作れる。
  提出物やSNSアカウント用にあると便利。
- READMEやデモ動画のURLが `gpsart.example.com` になるだけで審査の印象が変わる。
