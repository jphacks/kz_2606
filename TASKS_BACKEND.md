# バックエンド（2人）

前提: [DATA_MODEL.md](./DATA_MODEL.md) 4〜8章 / [TECH_PLAN.md](./TECH_PLAN.md) 7.5

## 構成

```
backend/
├── go.mod
├── cmd/server/                     エントリポイント。配線と go:embed だけ
└── internal/
    ├── domain/                     エンティティと不変条件。外部依存なし
    │   point.go  member.go  room.go  route.go  user.go  code.go  errors.go
    ├── usecase/                    アプリケーションロジック + 出力ポート
    │   port.go  room.go  session.go  auth.go
    ├── interface/
    │   rest/  router.go  room_handler.go  auth_handler.go  admin_handler.go
    │           middleware.go  static.go
    │   ws/    dto.go  client.go  handler.go  registry.go
    └── infrastructure/
        config/      config.go
        auth/        bcrypt.go  cookie.go
        persistence/ postgres.go  user_repository.go  room_repository.go
                     route_repository.go  point_repository.go  schema.sql
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
- bcrypt も Cookie 署名も `infrastructure/auth/` に閉じる

## 担当割り

| | 担当 | 持ちディレクトリ |
|---|---|---|
| **B1** | リアルタイム | `usecase/session.go` `interface/ws/` |
| **B2** | 認証・永続化・REST | `infrastructure/` `interface/rest/` `usecase/room.go` `usecase/auth.go` `cmd/` |
| **共同** | 最初に2人で決める | `internal/domain/` `internal/usecase/port.go` |

B2 の方が範囲が広いので、**認証は後述の⑤が通ってから**に回しています。

---

## 0. 最初に2人でやること（45分）— ここが全部の前提

### `internal/domain/`

```go
type Point struct { Lat, Lng float64; TS int64; Acc float32 }

type Member struct {
	ID          int64
	UserID      int64
	DisplayName string
	Color       string
	BannedAt    *time.Time
}

type Route struct {
	ID        int64
	MemberID  int64
	Points    []Point
	StartedAt time.Time
	EndedAt   *time.Time
}

type RoomCode string

type Room struct {
	Code        RoomCode
	ID          int64
	Title       string
	OwnerUserID int64
	ClosedAt    *time.Time
	Members     map[int64]Member
	Routes      map[int64]*Route
	Active      map[int64]int64   // memberID → 進行中の routeID
}
```

メソッド名も決める: `Join` / `Leave` / `UpdateMember` / `Ban` / `Unban` / `Close` / `Reopen` /
`StartRoute` / `AddPoints` / `EndRoute` / `DeleteRoute` / `Clone` / `CanWrite` / `IsOwner`

**`Clone()` は必須です。** snapshot は actor が所有している実体を外に出せないのでコピーを返します。
**`Active` を忘れないこと。** これが無いと「記録中か」が判定できません。

### `internal/usecase/port.go`

インターフェースだけ。実装は書かない。

```go
UserRepository   // users
RoomRepository   // rooms / members
RouteRepository  // routes（記録開始で作成、終了で ended_at）
PointRepository  // points のバルク保存（CopyFrom）
PasswordHasher   // 実装: infrastructure/auth/bcrypt.go
SessionCodec     // 実装: infrastructure/auth/cookie.go
Notifier         // 実装: interface/ws/registry.go
```

### `SessionService` のシグネチャも先に決める

```go
Join(ctx, code, memberID) (*domain.Room, error)   // ← 戻り値が snapshot の中身
StartRoute(ctx, code, memberID) (*domain.Route, error)
AppendPoints(ctx, code, routeID, pts []domain.Point) error
StopRoute(ctx, code, routeID) error
UpdateMember(ctx, code, m domain.Member) error
Leave(ctx, code, memberID)
```

**snapshot は `Join` の戻り値、他人への通知は `Notifier`。** こう分けると `usecase` が
接続もJSONも知らずに済みます。「誰が snapshot を送るのか」で迷いがちなので先に決めてください。

**この2つを push してから分かれる。** 以降は互いのファイルを見なくて済みます。

---

## 1. 縦に1本ずつ通す（層ごとに横に埋めない）

| 順 | 通すもの | 担当 | 確認 |
|---|---|---|---|
| ① | `GET /api/health` | B2 | `curl localhost:8080/api/health` |
| ② | `infrastructure/memory/` | B2 | B1 が Postgres を待たなくなる |
| ③ | `POST /api/rooms` → code が返る | B2 | `curl -XPOST localhost:8080/api/rooms -d '{"title":"test"}'` |
| ④ | `GET /ws` → **空の snapshot を返すだけ** | B1 | ブラウザのコンソールで接続して `snapshot` が来る |
| ⑤ | `start` → `points` → ブロードキャスト | B1 | **タブ2つで線が同期 ← 第1ゴール** |
| ⑥ | 認証一式 | B2 | `register` → `login` → `me` が通る |
| ⑦ | Postgres 一式 | B2 | 再起動して軌跡が戻る |
| ⑧ | `session.go` の flush（5秒ごと） | B1 | points が DB に入る |
| ⑨ | `route_end` / `closed` / `banned` | B1+B2 | owner 操作が全員に伝わる |
| ⑩ | `go:embed` + デプロイ | B2 | **⑤が通った時点で一度上げる** |

④で「空の snapshot を返すだけ」を先に通すのが肝です。**接続が確立すること**と
**ロジックが正しいこと**を分けて確認できます。

⑥の認証を後ろにしているのは、⑤まではブラウザのコンソールと curl で動作確認でき、
Cookie が無くても困らないからです。⑤が通ってから入れてください。

## 2. B1 の順番（リアルタイム）

1. `ws/dto.go` — タプル変換 + 全メッセージ型。**ここだけ `go test` を書く**
2. `ws/client.go` — `readPump` / `writePump` / **ping 30秒**
3. `ws/handler.go` — Cookie で本人特定 → member 確認 → snapshot
4. `usecase/session.go` — 部屋ごとに goroutine 1本。`Join` / `StartRoute` / `AppendPoints`
5. `ws/registry.go` — 部屋 → 接続集合。`Notifier` 実装。**送信者を除外**
6. `session.go` に `StopRoute` / `Leave` / flush を足す

`dto.go` のタプル変換はテストが数行で書けて、ここのバグはフロント側から原因が追いにくいので、
テストの費用対効果が一番高いところです。

## 3. B2 の順番（認証・永続化・REST）

1. `infrastructure/config/config.go`
2. **`infrastructure/memory/`** ← 最優先。B1 を解放する
3. `rest/room_handler.go` + `router.go` — create / get / join
4. `cmd/server/main.go` の配線
5. — **B1 と合流して⑤を確認** —
6. `infrastructure/auth/bcrypt.go` + `cookie.go` → `usecase/auth.go` → `rest/auth_handler.go` + `middleware.go`
7. `persistence/postgres.go` → `user_` → `room_` → `route_` → `point_repository.go`
8. `rest/admin_handler.go` — close / reopen / ban / route削除
9. `cmd/server/embed.go` + `rest/static.go`
10. デプロイ（Koyeb + Neon）

## 落とし穴

- **`Leave` で開いている route を自動で閉じる。** 忘れると `ended_at` が永久にNULLの
  route が溜まり、「記録中」のインジケータが消えなくなります。**一番踏みやすいバグ**
- **1点ずつINSERTしない。** 5秒ごとにまとめて `CopyFrom`
- **ping を消さない。** Cloudflare は無通信100秒で切る
- **クライアント申告の member ID を受け取らない。** Cookie から決める
- **ログイン失敗の理由を区別しない。** `invalid_credentials` に統一（ユーザー名の列挙を防ぐ）
- **`SESSION_SECRET` をコミットしない。** `openssl rand -base64 32` で生成
- **パスワードをログにもエラーメッセージにも出さない**
- **`domain.Room` は並行安全ではない。** 触るのは `session.go` の actor goroutine だけ
- **`routes` を記録開始時に同期で作る。** 非同期にすると `points` の外部キーが壊れる

## やらないこと

マイグレーションツール / ORM / PostGIS / 点のサーバー側フィルタ（フロントの仕事） /
パスワードリセット / レート制限 / セッションの即時失効
