# バックエンド（2人）

前提: [DATA_MODEL.md](./DATA_MODEL.md) 4〜7章 / [TECH_PLAN.md](./TECH_PLAN.md) 7.5

## 構成

```
backend/
├── go.mod
├── cmd/server/                     エントリポイント。配線と go:embed だけ
│   ├── main.go
│   ├── embed.go
│   └── dist/                       ← frontend のビルド出力
└── internal/
    ├── domain/                     エンティティと不変条件。外部依存なし
    │   point.go  member.go  room.go  code.go  errors.go
    ├── usecase/                    アプリケーションロジック + 出力ポート
    │   port.go  room.go  session.go
    ├── interface/                  外界との変換
    │   rest/  router.go  room_handler.go  static.go
    │   ws/    dto.go  client.go  handler.go  registry.go
    └── infrastructure/             技術的詳細
        config/      config.go
        persistence/ postgres.go  room_repository.go  point_repository.go  schema.sql
        memory/      room_repository.go
```

## 依存のルール（これだけは崩さない）

```
cmd/server  ─→ 全部（配線する唯一の場所）

interface ─→ usecase ─→ domain
                  ↑
          infrastructure（port を実装する側）
```

- **`usecase` から `infrastructure` を import しない。** `usecase/port.go` のインターフェースに対して書く
- **`domain` は何も import しない**（標準ライブラリだけ）
- `interface/ws/dto.go` だけが JSON のワイヤ形式を知る。`domain` に json タグを付けない

## 担当割り — 1ディレクトリ1人

| | 担当 | 持ちディレクトリ |
|---|---|---|
| **B1** | リアルタイム | `internal/usecase/session.go` `internal/interface/ws/` |
| **B2** | 永続化・配信 | `internal/infrastructure/` `internal/interface/rest/` `internal/usecase/room.go` `cmd/server/` |
| **共同** | 最初に2人で決める | `internal/domain/` `internal/usecase/port.go` |

## 最初に2人でやること（30分）— ここが全部の前提

1. **`internal/domain/` を書く。** エンティティだけなので30分で終わる
   - `Point{Lat, Lng, TS, Acc}` / `Member{ID, Name, Color}` / `Room{Code, Members, Tracks}` / `RoomCode`
   - `Room` のメソッド名を決める: `Join` / `Leave` / `AddPoints` / `UpdateMember`
2. **`internal/usecase/port.go` を書く。** インターフェースだけ。実装は書かない
   - `RoomRepository` — 部屋とメンバーの保存・復元
   - `PointRepository` — 点のバルク保存・読み出し
   - `Notifier` — 部屋のメンバーへ配る（B1 の `ws/registry.go` が実装）
3. **この2つをpushしてから分かれる。** 以降は互いのファイルを見なくて済む

## B1 の手順（リアルタイム）

1. `interface/ws/dto.go` — `Point` を `[lat, lng, ts, acc]` タプルに変換。メッセージ型（DATA_MODEL.md 5章）
2. `interface/ws/client.go` — read/write pump、**ping 30秒**、送信バッファ溢れたら切断
3. `interface/ws/registry.go` — 部屋コード → 接続集合。`usecase.Notifier` を実装。**送信者を除いて**ブロードキャスト
4. `interface/ws/handler.go` — upgrade → 1通目 `join` を検証 → `SessionService.Join`
5. `usecase/session.go` — **部屋ごとに goroutine 1本**を立てて `domain.Room` への操作を直列化。5秒ごとに `PointRepository` へ flush

**ゴール**: 2つのブラウザで同じ部屋に入って、互いの線がリアルタイムに出る（DBなしで）

## B2 の手順（永続化・配信）

1. `infrastructure/config/config.go` — `DATABASE_URL` / `PORT` / `ALLOW_ORIGIN`
2. `infrastructure/memory/room_repository.go` — **先にこれを作る。** B1 が DB を待たずに動ける
3. `interface/rest/room_handler.go` + `router.go` — `POST /api/rooms` / `GET /api/rooms/{code}` / `GET /api/health`
4. `infrastructure/persistence/postgres.go` — pgxpool 接続 + 起動時に `schema.sql` 実行
5. `persistence/room_repository.go` / `point_repository.go` — 保存は **`CopyFrom`**
6. `cmd/server/embed.go` + `interface/rest/static.go` — 埋め込みSPAの配信
7. **デプロイを通す（初日の夕方まで）** — Koyeb or Fly.io + Neon

**ゴール**: サーバー再起動しても軌跡が戻る。1つのURLでフロントもAPIも配信される

## 落とし穴

- **1点ずつINSERTしない。** 5秒ごとにまとめて `CopyFrom`
- **ping を消さない。** Cloudflare は無通信100秒で切る
- **`points` に外部キーを張らない。** メンバー保存が非同期なので `members` より先に点が来る
- **`DATABASE_URL` 空でも起動できるようにする。** `infrastructure/memory` に差し替えるだけ。デモ保険
- **自分の点は送信者にエコーバックしない。** 返すとフロントで線が二重になる
- **`domain.Room` は並行安全ではない。** 触るのは `session.go` の actor goroutine だけ

## やらないこと

認証 / マイグレーションツール / ORM / PostGIS / 点のサーバー側フィルタ（フロントの仕事）
