# データ構造仕様

認証・`routes`・部屋の締め切り・BAN まで反映した確定版。
後から決まる機能は「12. 拡張の作法」に、既存構造を壊さずに足す方針だけ書いてある。

関連: [TECH_PLAN.md](./TECH_PLAN.md) / [TASKS_BACKEND.md](./TASKS_BACKEND.md) / [TASKS_FRONTEND.md](./TASKS_FRONTEND.md)

---

## 1. 全体の規約

| 項目 | 決定 |
|---|---|
| 座標 | WGS84。緯度 `lat` → 経度 `lng` の順（**GeoJSONは逆順**なので注意） |
| 時刻（ワイヤ） | 点列は **unixミリ秒の数値**。それ以外は **RFC3339 文字列** |
| 時刻（DB） | `TIMESTAMPTZ`。アプリ側は常にUTCで扱う |
| 距離 | メートル / 速度 m/s |
| 色 | `#rrggbb` の6桁16進。短縮形・名前付き色・alphaは不可 |
| ID | すべて `BIGSERIAL`（int64）。JSONでは**数値**で送る |
| 状態フラグ | boolean ではなく `*_at TIMESTAMPTZ` の NULL / 非NULL で表す |
| JSONキー | 点列だけ短縮（`pts`）。他は読みやすさ優先の camelCase |
| 真実のソース | **Postgres**。route・point はトランザクションで保存し、snapshot も DB から読む |
| 認証 | **ログイン必須。** 匿名参加は現時点では無し（12章に拡張方針） |

> JSONの数値は JavaScript で `2^53` までしか安全に扱えないが、`BIGSERIAL` の連番が
> そこに到達することはないので、IDを文字列化する必要はない。

---

## 2. ID と識別子

| 種別 | 形式 | 発行元 | 用途 |
|---|---|---|---|
| `users.id` ほか全ID | `BIGSERIAL`（int64） | **DB** | 内部参照 |
| `rooms.code` | 6文字。`A-HJ-NP-Z2-9`（`I` `O` `0` `1` を除く32文字） | サーバー | **URL・QRで共有するのはこれ。`id` は外に出さない** |
| セッション | HttpOnly Cookie（HMAC-SHA256 署名） | サーバー | 本人の特定。WSのハンドシェイクにも自動で乗る |
| `username` | 3〜16文字 `[a-z0-9_]`、**小文字で保存**、UNIQUE | ユーザー入力 | ログイン |

- 部屋コードの正規表現は**サーバーとクライアントで同一**: `^[A-HJ-NP-Z2-9]{6}$`
- 入力は常に `toUpperCase()` してから判定する
- `rooms.id` を URL に出さないこと。連番なので他人の部屋を列挙できてしまう
- **`Authorization` ヘッダは使わない。** ブラウザの `WebSocket` にカスタムヘッダは付けられないため、
  セッションは必ず Cookie に載せる

---

## 3. Point（座標点）

点は**4要素のタプル**で表現する。オブジェクトにしない（キー名が点数分だけ繰り返されるため）。

```ts
type Point = [lat: number, lng: number, ts: number, acc: number];
```

| 添字 | 意味 | 単位 |
|---|---|---|
| `0` | 緯度 | 度 |
| `1` | 経度 | 度 |
| `2` | 取得時刻 | unixミリ秒 |
| `3` | 測位精度 | メートル |

Go側は構造体で持ち、`interface/ws/dto.go` でタプルに変換する。
**`domain.Point` に json タグを付けない。** ワイヤ形式を変えるときは dto だけ直す。

```go
type Point struct {
	Lat, Lng float64
	TS       int64   // unix milliseconds
	Acc      float32
}
```

ワイヤ上の見た目:

```json
[43.060712, 141.354451, 1761278400000, 8.2]
```

### 点の採否ルール（クライアント側で確定させる）

サーバーは点をフィルタしない。**捨てる判断はすべてクライアント**（`frontend/src/lib/geo.ts`）。

| 条件 | 動作 | 理由 |
|---|---|---|
| `acc > 30` | 捨てる | 粗い測位は線を暴れさせる |
| 直前の点から `< 5m` | 捨てる | 立ち止まったときのジッターで団子になる |
| 速度 `> 14 m/s`（≒50km/h） | 捨てる | 基地局測位の飛び |
| 直前の点から `> 30秒` | **捨てない**。距離・速度判定もしない | ギャップ明けの最初の点を失わないため |

---

## 4. DBスキーマ

`backend/internal/infrastructure/persistence/schema.sql`。起動時に毎回そのまま実行される。
**PostGISは使わない。** 空間関数も空間インデックスも必要ない。

```sql
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,   -- 小文字で保存
    password_hash TEXT NOT NULL,          -- bcrypt
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rooms (
    id            BIGSERIAL PRIMARY KEY,
    code          TEXT NOT NULL UNIQUE,   -- 6文字。共有するのはこれ
    title         TEXT NOT NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users (id),
    is_private    BOOLEAN NOT NULL DEFAULT false,
    closed_at     TIMESTAMPTZ,            -- NULL = 進行中
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE rooms ADD COLUMN IF NOT EXISTS is_private BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS members (
    id           BIGSERIAL PRIMARY KEY,
    room_id      BIGINT NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users (id),
    display_name TEXT NOT NULL,
    color        TEXT NOT NULL,           -- #rrggbb
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    banned_at    TIMESTAMPTZ,             -- NULL = 参加中
    UNIQUE (room_id, user_id)             -- 1部屋1ユーザー1行
);

CREATE TABLE IF NOT EXISTS routes (
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    member_id  BIGINT NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at   TIMESTAMPTZ               -- NULL = 記録中
);

CREATE INDEX IF NOT EXISTS routes_room ON routes (room_id);

-- 部屋の中で記録中（= 編集可）の route は同時に1本だけ。
-- 複数人が同時に start を叩いても2本目が作られない。
CREATE UNIQUE INDEX IF NOT EXISTS routes_one_recording_per_room
    ON routes (room_id) WHERE ended_at IS NULL;

CREATE TABLE IF NOT EXISTS points (
    id          BIGSERIAL PRIMARY KEY,
    route_id    BIGINT NOT NULL REFERENCES routes (id) ON DELETE CASCADE,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    accuracy    REAL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS points_route_time ON points (route_id, recorded_at);
```

### 設計の理由

**`routes` = 1メンバーの1回の記録。** 「記録開始」で1行作り、「記録終了」で `ended_at` を埋める。
これにより `points` は `route_id` だけで辿れ、`member_id` を持たせる必要がない。

**外部キーを張れる理由。** `routes` は記録開始時に**同期で**作るので、点より必ず先に存在する。
`rooms → members → routes → points` が `ON DELETE CASCADE` で繋がるので、部屋を消せば全部消える。

**`routes.room_id` は `member_id` から辿れるので冗長**だが意図的に残している。
「部屋の全軌跡を読む」ときに `members` をJOINせずに済む。

**`UNIQUE (room_id, user_id)`** により再入室・リロードが自然に冪等になる。

### 列を追加するとき

`schema.sql` は起動時に毎回流れるので、**冪等なDDLだけを追記する**。

```sql
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS goal JSONB;
```

`CREATE TABLE` を書き換えるのではなく `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` を足す。
マイグレーションツールは入れない。

---

## 5. REST API

### エラーの形（全エンドポイント共通）

```json
{"error": {"code": "username_taken", "message": "そのユーザー名は使われています"}}
```

フロントは `code` で分岐し、`message` をそのまま表示する。

### 認証

| メソッド・パス | リクエスト | 成功 | エラー |
|---|---|---|---|
| `POST /api/auth/register` | `{username, password}` | `200 {user, room}` + Cookie | `400 invalid_username` / `400 invalid_password` / `409 username_taken` |
| `POST /api/auth/login` | `{username, password}` | `200 {user, room}` + Cookie | `400 invalid_format` / **`401 invalid_credentials`** |
| `POST /api/auth/logout` | — | `204` | — |
| `GET /api/auth/me` | — | `200 {user, room}` | `401 unauthorized` |

登録時の `username` は3〜16文字の英数字または `_`（大文字は小文字に正規化）、
`password` は8〜72バイトとする。ログインと登録はいずれも `{username, password}` を受け取る。

```json
// user の形
{"user": {"id": 1, "username": "taro"},
 "room": {"id": 7, "code": "KZ26AB", "title": "taroのルーム",
          "ownerMemberId": 1, "isPrivate": true, "closedAt": null}}
```

> **`invalid_credentials` は統一する。** 「ユーザーが存在しない」と「パスワードが違う」を
> 区別するとユーザー名を列挙できてしまう。実装も分岐が消えて楽になる。

> **登録成功時にそのままログインさせる**（Cookie を返す）。
> 分けるとユーザーがパスワードを2回打つことになる。

> **アカウント登録時に個人ルームを作る。** タイトルは `{username}のルーム`、
> owner と member は登録した本人だけ。`is_private` により他ユーザーのjoinと
> WebSocket接続を拒否する。通常の `POST /api/rooms` で作るルームは公開のまま。

> **`GET /api/auth/me` はフロントの起動時に必ず叩く。**
> これが無いとリロードでログイン状態が消える（Cookieは残っているのに自分が誰か分からない）。
> 登録・ログイン・`/me` はいずれも個人ルームを含めて返すため、別端末やリロード後も部屋コードを復元できる。
> **個人ルームが無いアカウント（この仕組みより前に登録したもの）は、ログインまたは `/me` の
> 時点でその場で作る。** 「無い」をエラーにすると既存アカウントがログインできなくなる。
> 個人ルームは1ユーザーにつき1つで、`rooms_one_private_room_per_owner` の部分ユニーク索引で
> 同時ログイン時の二重作成を防ぐ。

### Cookie

```
Set-Cookie: session=<HMAC署名付き>; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=604800
```

- 中身は `userID` と失効時刻を HMAC-SHA256 で署名したもの。**セッションテーブルは作らない**
- 鍵は `SESSION_SECRET`（env）。**サーバー再起動でログアウトしない**のが開発中に効く
- 失効の即時無効化はできない。ハッカソンでは割り切る

### 部屋

| メソッド・パス | リクエスト | 成功 | エラー |
|---|---|---|---|
| `POST /api/rooms` | `{title}` | `200 {room}` | `401 unauthorized` / `400 invalid_title` |
| `GET /api/rooms/{code}` | — | `200 {room}` | `404 room_not_found` |
| `POST /api/rooms/{code}/join` | `{displayName, color}` | `200 {member, room}` | `401` / `404` / `400 invalid_color` / `403 banned` / `403 private_room` / `400 invalid_display_name` |

```json
// room の形
{"room": {"id": 7, "code": "KZ26AB", "title": "大通公園に絵を描く",
          "ownerMemberId": 1, "isPrivate": false, "closedAt": null}}
```

- `POST /api/rooms` は**レスポンスに `code` を含める**。作成者は即QRを出す必要がある
- 作成者は自動で member になる（`displayName` は `username`、色はパレットの先頭 `#0e7861`）
- 個人ルームは `isPrivate: true`。本人以外のjoinおよびWS接続を拒否する
- `GET /api/rooms/{code}` は存在確認だけ。**軌跡は返さない**（それは `snapshot` の仕事）
- `POST .../join` は**冪等**。すでに参加済みなら既存の member を返す
- `title` は前後の空白を除いて1〜40文字 / `displayName` は前後の空白を除いて1〜16文字

### 管理（owner のみ）

| メソッド・パス | 成功 | エラー |
|---|---|---|
| `POST /api/rooms/{code}/close` | `200 {closedAt}` | `403 forbidden` |
| `POST /api/rooms/{code}/reopen` | `200` | `403 forbidden` |
| `POST /api/rooms/{code}/members/{memberId}/ban` | `204` | `403 forbidden` / `400 cannot_ban_owner` / `404 member_not_found` |
| `DELETE /api/rooms/{code}/members/{memberId}/ban` | `204` | 同上 |
| `DELETE /api/rooms/{code}/routes/{routeId}` | `204` | `403 forbidden` / `404 route_not_found` |

- route の削除は**本人または owner**。それ以外は owner のみ
- **状態を変えるのはREST、全員への通知はWS。** 実装は
  `usecase.RoomService.Close()` が `Notifier.NotifyClosed()` を呼ぶ形になる

### その他

```
GET  /api/health          → {"ok": true}
GET  /ws?room={code}      → WebSocket upgrade
GET  /*                   → 埋め込み静的ファイル / SPA（画面パスは index.html。未知のassetと /api・/ws は404）
```

---

## 6. WebSocket プロトコル

```
GET /ws?room=KZ26AB
```

- **本人は Cookie から特定する。** クライアント申告のIDは一切受け取らない
- その部屋の member でなければ `403` で切断（先に `POST .../join` を叩かせる）
- BAN されている場合も `403`
- 接続直後にサーバーが `snapshot` を送る。クライアントからのハンドシェイクは**不要**
- サーバー→クライアントの `ping` は30秒ごと（Cloudflareの無通信100秒切断対策）
- サーバー側の読み取り上限は **64KB/メッセージ**

### Client → Server

| `t` | ペイロード | 備考 |
|---|---|---|
| `start` | — | 記録開始。サーバーが `routes` を1行作る |
| `points` | `{pts: Point[]}` | 1通あたり最大500点（64KB上限との兼ね合い） |
| `stop` | — | 記録終了。`ended_at` を埋める |
| `member` | `{displayName?, color?}` | 名前・色の変更 |

```json
{"t": "start"}
{"t": "points", "pts": [[43.0607,141.3544,1761278400000,8.2],[43.0608,141.3545,1761278403000,7.5]]}
{"t": "stop"}
{"t": "member", "color": "#00e5ff"}
```

- `start` を送らずに `points` を送ったら `error / no_active_route`
- 締められた部屋で `start` / `points` を送ったら `error / room_closed`
- **すでに誰かが記録中の部屋で `start` を送ったら `error / route_in_use`。**
  部屋の中で編集できる route は常に1本だけなので、前の人が `stop` するまで待つ

### Server → Client

| `t` | ペイロード | いつ |
|---|---|---|
| `snapshot` | `{room, you, members, routes}` | 接続直後に1回。**現在の絵をまるごと** |
| `route` | `{route: {id, member, color, startedAt}}` | 誰かが記録を開始した |
| `points` | `{route, pts}` | 増分。**自分の点は返ってこない** |
| `route_end` | `{route, member, endedAt}` | 誰かが記録を終了した |
| `member` | `{member}` | 参加 or 名前/色の変更 |
| `leave` | `{member}` | 全接続が切れた（member単位） |
| `closed` | `{closedAt, by}` | owner が締めた |
| `reopened` | `{by}` | owner が再開した |
| `banned` | `{member, by}` | 誰かがBANされた |
| `unbanned` | `{member, by}` | BAN解除 |
| `route_deleted` | `{route}` | route が削除された |
| `error` | `{code, message}` | 操作が拒否された |

### snapshot の全体像

```json
{
  "t": "snapshot",
  "room": {
    "id": 7, "code": "KZ26AB", "title": "大通公園に絵を描く",
    "ownerMemberId": 1, "isPrivate": false, "closedAt": null
  },
  "you": { "id": 3, "displayName": "たろう", "color": "#ff2d78", "isOwner": false },
  "members": [
    { "id": 1, "displayName": "はなこ", "color": "#0e7861", "online": true,  "banned": false, "recording": true  },
    { "id": 3, "displayName": "たろう", "color": "#ff2d78", "online": true,  "banned": false, "recording": false },
    { "id": 5, "displayName": "じろう", "color": "#ffe600", "online": false, "banned": true,  "recording": false }
  ],
  "routes": [
    { "id": 12, "member": 1, "color": "#0e7861",
      "startedAt": "2026-10-24T05:00:00Z", "endedAt": "2026-10-24T05:10:00Z",
      "meters": 842.5, "elapsedMs": 600000,
      "pts": [[43.0607,141.3544,1761278400000,8.2]] },
    { "id": 14, "member": 1, "color": "#0e7861",
      "startedAt": "2026-10-24T05:30:00Z", "endedAt": null,
      "meters": 0, "elapsedMs": 1800000,
      "pts": [] }
  ]
}
```

### その他のメッセージ例

```json
{"t":"route",         "route": {"id": 15, "member": 3, "color": "#ff2d78", "startedAt": "2026-10-24T06:00:00Z"}}
{"t":"points",        "route": 15, "pts": [[43.0611,141.3550,1761278410000,6.1]]}
{"t":"route_end",     "route": 15, "member": 3, "endedAt": "2026-10-24T06:30:00Z"}
{"t":"member",        "member": {"id": 3, "displayName": "たろう", "color": "#00e5ff", "online": true, "banned": false, "recording": false}}
{"t":"leave",         "member": 5}
{"t":"closed",        "closedAt": "2026-10-24T06:00:00Z", "by": 1}
{"t":"banned",        "member": 5, "by": 1}
{"t":"route_deleted", "route": 15}
{"t":"error",         "code": "room_closed", "message": "この部屋は締め切られています"}
```

### 不変条件

- **点は `route` 単位で送る。** member は route から引けるので重複させない
- **自分の点はエコーバックされない。** クライアントは送信前にローカルで描画する。
  ここを変えると線が二重になる
- **色は `members[]` が正**、`routes[].color` は描画用のコピー。
  色変更時は `member` メッセージが来るのでズレない
- **`members` には退出済み・BAN済みも全員含める。**
  そうしないと残っている route の色が引けなくなる
- **`leave` と `route_end` は別物。**
  `route_end` = 記録を止めた（まだ部屋にいる） / `leave` = 接続が切れた
- **未知の `t` は両側で黙って無視する。** これがプロトコル拡張の唯一のルール。
  新しいメッセージ種別は古いクライアントを壊さずに足せる
- **経過時間と距離は `route` に紐づける。部屋には持たせない。**
  UIに出すのは「この経路を何分で何km歩いたか」なので、集計の単位も route に揃える。
  部屋全体の合計が要るときは、クライアントが `routes[]` の分を足す
  （`members[]` から人数を数えるのと同じ扱い）
  - `routes[].meters` はその route の累積距離。
    点の間が `GAP_MS`（30秒）を**超えて**空いた区間は距離に数えない
    （3章の `> 30秒` と同じ境界。ちょうど30秒は数える）。
    クライアントの `geo.ts` と同じルールなので、**片方だけ変えると数字がズレる**
  - `routes[].elapsedMs` は `startedAt` から、記録を止めてあれば `endedAt` まで。
    **止めたら止まる**
  - 記録中の route では `elapsedMs` は snapshot を作った瞬間の値なので、
    **クライアントは `startedAt` を基準に自分で時計を進める**
  - そのため `route` / `route_end` のメッセージも `startedAt` / `endedAt` を運ぶ。
    snapshot 以外で知った route でも時計を動かせるようにするため
  - BAN されたメンバーの route も `routes[]` には入る（色を引けなくなるため）。
    合計を出すときに含めるかは、`members[].banned` を見てクライアントが決める

### サーバー側のバリデーション

クライアント申告値は信用しない。

| 項目 | 不正時の扱い |
|---|---|
| member ID | **受け取らない。** Cookie から決める |
| `displayName` が空 | 変更しない |
| `displayName` が17文字以上 | 16文字に切り詰め（rune単位） |
| `color` が `^#[0-9a-fA-F]{6}$` に合わない | 変更しない |

---

## 7. 権限と状態遷移

### 権限

| 操作 | できる人 |
|---|---|
| 部屋を締める / 再開する | owner のみ |
| メンバーをBAN / 解除 | owner のみ。**owner自身はBANできない** |
| route を削除 | 本人 または owner |
| 名前・色の変更 | 本人のみ |

### 部屋の状態

```
open ──close──> closed ──reopen──> open
```

| | open | closed |
|---|---|---|
| 既存のWS接続 | — | **切らない。** 絵を見続けられる |
| `start` / `points` | 可 | **拒否**（`error / room_closed`） |
| 新規の join・WS接続 | 可 | **可**（閲覧のみ = 公開ギャラリー） |
| 記録中だった route | — | サーバーが自動で `ended_at` を埋める |

「更新を打ち切る」＝書き込み禁止・閲覧は自由。QRをそのまま審査員に渡せる。

### メンバーの状態

```
active ──ban──> banned ──unban──> active
```

- BANは `banned_at` を立てるだけ。**DBから軌跡は消さない**（誤BANを戻せる）
- フロントは `banned: true` のメンバーの route を**非表示**にする
- 完全に消したいなら `DELETE .../routes/{routeId}` を別途叩く
- BAN された本人には `error / banned` を送って切断する

### route の状態

```
recording（ended_at IS NULL） ──stop──> ended
```

**1つの部屋は route を何本でも持つ。ただし編集できるのは常に1本だけ。**

| | 本数 |
|---|---|
| 部屋が持つ route | 何本でも（記録のたびに1本増える） |
| 記録中（`ended_at IS NULL`）の route | **部屋全体で1本まで** |

- 編集可 = 記録中の route。`points` が追記されるのはこの1本だけ
- 記録済みの route は**読み取り専用**。消すなら `DELETE .../routes/{routeId}`
- 誰かが記録中のあいだ、他のメンバーは `start` できない（`error / route_in_use`）。
  **`members[].recording` が `true` になるのも同時に1人まで**
- DB側は `routes_one_recording_per_room` の部分ユニーク索引で保証する。
  同時に `start` が飛んできても2本目は作られない

> **`leave` の時点で開いている route を自動で閉じること。**
> これを忘れると `ended_at` が永久にNULLの route が溜まり、
> 「記録中」のインジケータが消えなくなる。一番踏みやすいバグ。
> **編集可が部屋で1本なので、閉じ忘れると他の誰も記録を始められなくなる。**

---

## 8. サーバー内のセッション・接続管理

現在の実装では、route と point の永続データは Postgres に保持する。ルームの軌跡を保持する
インメモリHubやactorは使わず、`start`・`points`・`stop` の各操作はルーム行をロックする
トランザクション内で処理する。点はバッチで `CopyFrom` し、保存が成功してから他の接続へ通知する。
新規接続時のsnapshotもDBから読み込む。

```text
Postgres
 ├ rooms / members / routes / points   永続データと記録状態
 └ routes_one_recording_per_room       部屋内の記録中routeを1本に制限

Registry                              interface/ws
 └ conns map[roomCode]map[*client]struct{}  接続と複数タブの管理のみ

client
 ├ conn       *websocket.Conn
 ├ out        chan []byte              容量64。溢れたら切断
 ├ memberID   int64                    Cookieで認証したmember
 └ onLeave    func()                   最後の接続が切れた際にrouteを閉じる
```

### 操作と不変条件

1. `start` / `points` / `stop` は同じ部屋行の `FOR UPDATE` ロックを取る。記録開始時に部屋内のactive routeを調べ、二重開始は `route_in_use` で拒否する。部分ユニーク索引も最後の防衛線として維持する。
2. `points` は現在のmemberのactive routeにだけ追加する。routeが無ければ `no_active_route` を返す。
3. 点列は1メッセージ500点以下。1回のバッチ保存が成功した後に限り、送信者以外へ `points` を配信する。送信者は自分の線をローカルで描画する。
4. 同一 `memberID` の接続が複数ありうる（複数タブ）。最後の接続が切れた時だけ `leave` を通知し、記録中routeがあれば終了する。この後処理は他接続への配信をブロックしないよう非同期で行う。
5. `snapshot` はDBから読み込む。インメモリのroute/point複製や定期flushとの整合性管理は行わない。

---

## 9. クライアント状態

### zustand ストア（`frontend/src/store.ts`）

```ts
{
  user:      { id: number; username: string } | null   // GET /api/auth/me で復元
  room:      Room | null
  me:        Member | null                             // 自分の member
  status:    "idle" | "connecting" | "open" | "closed"
  members:   Record<number, Member>                    // online / banned / recording 込み
  routeOwner:Record<number, number>                    // routeId → memberId
  stats:     Record<number, { points: number; meters: number }>  // memberId 単位
  recording: boolean
  mock:      boolean
  wakeLockHeld: boolean
  error:     string | null
}
```

### 軌跡は zustand に入れない（重要）

点列は `useTracks` の `useRef` が持つ。**Reactの状態にしない。**
毎秒頂点が増えるため、状態に載せると毎秒全体が再レンダリングされる。

```ts
Map<routeId, {
  group:    L.LayerGroup   // 地図に載っているレイヤ
  segments: L.Polyline[]   // ギャップ・点数上限で分割された区間
  points:   Point[]        // 統計・書き出し用
  meters:   number         // 累積距離。毎回再計算しない
  color:    string
  memberId: number
}>
```

- **キーは `routeId`。** route 1本 = ポリラインの束。route 境界で線が切れるのが自然になる
- さらに `GAP_MS`（30秒）以上空いたら、同じ route 内でもセグメントを分ける
- **1000点ごとにもセグメントを切る。** Leaflet の Canvas レンダラはレイヤ単位で
  画面との交差判定をするので、小さい polyline が並んでいると画面外を丸ごとスキップできる
- 色の変更は `memberId` から `routeId` を引いて全セグメントに `setStyle()`
- 再レンダリングが走るのは `stats` 更新時だけ

### 描画の使い分け

| 場面 | API | 理由 |
|---|---|---|
| `snapshot` の一括投入 | `setLatLngs()` | 1回しか再描画しない。`addLatLng` のループは固まる |
| 増分（1〜数点） | `addLatLng()` | そのまま伸ばす |

---

## 10. 永続キー

| 場所 | キー | 値 | 寿命 |
|---|---|---|---|
| Cookie | `session` | HMAC署名付き userID | 7日 |
| IndexedDB (idb-keyval) | `outbox:<ROOMCODE>` | `{routeId, pts}[]` | 送信成功で削除。上限20,000点 |
| URL クエリ | `?room=<CODE>` | 部屋コード | QR・共有リンクからの直接参加 |

`outbox` はWS切断中の点を保持する。**`routeId` と一緒に積む**こと。
再接続後に別の route が始まっていると、点の所属先が分からなくなる。

再接続時に500点ずつ送り直し、途中で失敗したら残りを書き戻す。

再接続snapshotに同じmemberの記録中routeがあればそのrouteを再開する。
routeが無い場合は切断中にサーバー側で終了済みなので、新しいrouteを開始する。
古いrouteのoutboxはroute IDが一致しないため新しいrouteへは送らず、容量上限まで保持する。

> **再送中は直送を止める。** `outbox` の吐き出しは非同期なので、その隙間に
> 新しい点を直送すると**時刻が逆行した点列**が他クライアントに届く。
> `draining` フラグを立てて、吐き出し中は必ず `outbox` に積む。

---

## 11. 定数一覧（連動しているものに注意）

| 定数 | 値 | 場所 | 連動相手 |
|---|---|---|---|
| `MAX_ACCURACY_M` | 30 | `frontend/src/lib/geo.ts` | — |
| `MIN_DISTANCE_M` | 5 | 同 | **点の総数を直接決める。** 1mにすると5倍 |
| `MAX_SPEED_MPS` | 14 | 同 | — |
| `GAP_MS` | 30,000 | 同 | `TrackBuffer` のセグメント分割、距離計算の除外。**`domain.GapMillis` と同値に保つ** |
| `GapMillis` | 30,000 | `backend/internal/domain/distance.go` | `snapshot.routes[].meters` の算出。**`GAP_MS` と連動** |
| `SEGMENT_MAX_POINTS` | 1,000 | `frontend/src/lib/track_buffer.ts`（`maxTrackSegmentPoints`） | 描画性能。1本の Leaflet ポリラインの上限。超えたら確定して次のセグメントへ |
| 間引きの許容値 | 5m | `frontend/src/lib/geo.ts`（`trackSimplificationToleranceMeters`） | **`MIN_DISTANCE_M` と同値に保つ。** これを採否より大きくすると、わざわざ残した点を描画で落とすことになる |
| `FLUSH_MS` | 3,000 | `frontend/src/hooks/useRecorder.ts` | 送信頻度 = `stats` の更新頻度 |
| `CHUNK` | 500 | `frontend/src/lib/ws.ts` | **`maxMessageSize` 64KB を超えない上限** |
| `MAX`（outbox） | 20,000 | `frontend/src/lib/outbox.ts` | — |
| `pingPeriod` | 30s | `backend/internal/interface/ws/client.go` | Cloudflareの無通信100秒切断。**これより短く保つ** |
| `pongWait` | 70s | 同 | `pingPeriod` の2倍強 |
| `maxMessageSize` | 65,536 | 同 | **`CHUNK` と連動。片方だけ変えない** |
| `outBuffer` | 64 | 同 | 溢れると遅いクライアントを切断する閾値 |
| `flushInterval` | 5s | `backend/internal/usecase/session.go` | DB書き込み頻度 |
| `bcrypt cost` | 10（既定） | `backend/internal/infrastructure/auth/` | テストでは必ずダミーに差し替える |
| セッション有効期限 | 7日 | 同 | |

### 規模の目安

| | |
|---|---|
| 1人1時間（5m間隔・歩行1.4m/s） | 約1,000点 |
| 5人×2時間 | 約10,000点 ≒ snapshot 450KB / DB 1MB |
| **最初に壊れる箇所** | **snapshot のサイズ（約5万点 / 2MB）** |

Postgresは実質壊れない。心配する順番は snapshot → Leaflet の再投影 → メモリ → DB。

---

## 12. 拡張の作法（後から決まるもの）

**以下はコードに入っていない。** 既存構造を壊さずに足す方針だけ決めてある。

### 共通ルール

| 対象 | やり方 |
|---|---|
| 新しいWSメッセージ | 新しい `t` を足すだけ。未知の `t` は両側で無視されるので古いクライアントは壊れない |
| DBの列追加 | `schema.sql` に `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` を**追記** |
| 形が未確定なもの | `JSONB` 列1本で持つ。スキーマ変更なしに中身を変えられる |
| 点から導出できるもの | **保存しない**。クライアントで計算する |

### 匿名参加を戻す場合

ログイン必須をやめるなら、変更は3箇所。

```sql
ALTER TABLE members ALTER COLUMN user_id DROP NOT NULL;      -- NULL = 匿名
ALTER TABLE members ADD COLUMN IF NOT EXISTS client_token TEXT UNIQUE;
```

- `client_token` は推測不可な乱数（16バイトのhex）。クライアントが localStorage に保存する
- `members.id` は連番なので、これを localStorage に入れるとなりすませる。必ずトークンを使う
- `UNIQUE (room_id, user_id)` は NULL を重複扱いしないので、匿名は何人でも入れる
- join を「Cookie があればそれ、無ければ `client_token`」の優先順位にする

### リプレイのタイムラプス

**追加データ構造は不要。** `routes[].pts` に `ts` が入っているので、時刻でソートして再生するだけ。
`routes[].startedAt` / `endedAt` があるので「記録セッション単位の再生」もできる。

点数が多くて初期ロードが重くなったら、`GET /api/rooms/{code}/routes` を足して
`simplify()` 済みの軌跡を返す。**表示精度は画面解像度で決まる**（ズーム16で1px ≒ 2.4m）ので、
許容誤差3mで間引いても見た目は変わらない。

### お題レイヤー（目標の文字・図形）

形が決まっていないので `JSONB` で持つ。

```sql
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS goal JSONB;
```

```json
{
  "kind": "polyline",
  "title": "大通公園に『JP』",
  "strokes": [[[43.0607,141.3544],[43.0611,141.3550]], [[43.0615,141.3560]]],
  "assignments": { "3": [0, 2] },
  "toleranceM": 20
}
```

達成率は**クライアントで計算する**。WSには `{"t":"goal","goal":{...}}` を足す。

### 塗りつぶしグリッド

**永続化しない。** 点から導出する。セルIDの決め方だけ先に固定しておく。

```ts
const CELL_DEG = 0.00045;  // ≒50m（緯度方向）
const cellId = (p: Point) =>
  `${Math.floor(p[0] / CELL_DEG)}:${Math.floor(p[1] / CELL_DEG)}`;
```

固定グリッドなので**全員が同じ点から同じセルIDを出す**。クライアント集計とサーバー集計が一致する。

### 画像・GeoJSON書き出し

`useTracks` の `snapshot()` から作る。

```ts
// GeoJSON は [lng, lat] 順。Point タプルと逆なので必ず入れ替える
{
  type: "FeatureCollection",
  features: [{
    type: "Feature",
    geometry: { type: "LineString", coordinates: pts.map(p => [p[1], p[0]]) },
    properties: { routeId, memberId, displayName, color }
  }]
}
```

### まだ決めていないこと

| 項目 | メモ |
|---|---|
| 自分が参加した部屋の一覧 | `GET /api/rooms` が必要。マイページを作るなら |
| パスワード変更・リセット | 無い。当日忘れた人はロックアウトされる。使い捨て前提のアカウントとして運用する |
| セッションの即時失効 | 署名Cookieなので不可。必要なら `users.session_epoch` を足して署名に混ぜる |
| レート制限 | ログインに無し。公開運用するなら必要 |
| 部屋の有効期限・自動削除 | 今は無期限。`rooms` を消せばCASCADEで全部消える |
| 色の重複 | 許している。UIで使用済みを薄く表示する程度 |
| 1部屋あたりの人数上限 | 無し |
