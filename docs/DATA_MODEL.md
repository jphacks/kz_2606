# データ構造仕様

実装済みの構造は**確定**。後から決まる機能は「11. 拡張の作法」に、
既存構造を壊さずに足すための方針だけ書いてある（コードには未反映）。

関連: [TECH_PLAN.md](./TECH_PLAN.md)（設計方針） / [DEVELOP.md](./DEVELOP.md)（開発手順）

---

## 1. 全体の規約

| 項目 | 決定 |
|---|---|
| 座標 | WGS84。緯度 `lat` → 経度 `lng` の順（**GeoJSONは逆順**なので注意） |
| 時刻（ワイヤ） | **unixミリ秒の数値**。文字列にしない |
| 時刻（DB） | `TIMESTAMPTZ`。アプリ側は常にUTCで扱う |
| 距離 | メートル |
| 速度 | m/s |
| 色 | `#rrggbb` の6桁16進（大文字小文字どちらも可）。短縮形・名前付き色・alphaは不可 |
| JSONキー | 短縮形（`t`, `pts`）。点数が数千になるため |
| 真実のソース | **サーバーのメモリ**（`Room.tracks`）。DBは後追いの保存先 |

---

## 2. ID とコード

| 種別 | 形式 | 生成元 | 例 |
|---|---|---|---|
| 部屋コード | 6文字。`A-HJ-NP-Z2-9`（`I` `O` `0` `1` を除く32文字） | サーバー `NewRoomCode()` | `KZ26AB` |
| メンバーID | 16文字の英数字（32文字まで許容） | **クライアント**が生成し localStorage に保存 | `a3f1c8d920b74e55` |

- 部屋コードの正規表現は**サーバーとクライアントで同一**:
  `^[A-HJ-NP-Z2-9]{6}$`（`backend/internal/domain/code.go`、`frontend/src/App.tsx` 等の `CODE_RE`）
- 入力は常に `toUpperCase()` してから判定する。
- **メンバーIDは端末に紐づく**。同じ端末からの再入室は同一人物として扱われ、軌跡が継続する。
  別端末・別ブラウザからは別人になる。これは仕様（認証を作らないため）。

---

## 3. Point（座標点）— 最重要の決定

点は**4要素のタプル**で表現する。オブジェクトにしない（キー名が点数分だけ繰り返されるため）。

```ts
type Point = [lat: number, lng: number, ts: number, acc: number];
```

| 添字 | 意味 | 単位 |
|---|---|---|
| `0` | 緯度 | 度 |
| `1` | 経度 | 度 |
| `2` | 取得時刻 | unixミリ秒 |
| `3` | 測位精度 | メートル（`GeolocationCoordinates.accuracy`） |

Go側は構造体で持ち、`MarshalJSON` / `UnmarshalJSON` でタプルに変換する
（`backend/internal/interface/ws/dto.go`）。**ワイヤ形式を変えるときはこの2メソッドだけ直せばよい**。

```go
type Point struct {
	Lat float64
	Lng float64
	TS  int64   // unix milliseconds
	Acc float32
}
```

ワイヤ上の見た目:

```json
[43.060712, 141.354451, 1761278400000, 8.2]
```

### 点の採否ルール（クライアント側で確定させる）

サーバーは点をフィルタしない。**捨てる判断はすべてクライアント**（`frontend/src/lib/geo.ts` の `PointFilter`）。
サーバーに届いた点は無条件で軌跡に入る。

| 条件 | 動作 | 理由 |
|---|---|---|
| `acc > 30` | 捨てる | 粗い測位は線を暴れさせる |
| 直前の点から `< 5m` | 捨てる | 立ち止まったときのジッターで団子になる |
| 速度 `> 14 m/s`（≒50km/h） | 捨てる | 基地局測位の飛び |
| 直前の点から `> 30秒` | **捨てない**。距離・速度判定もしない | ギャップ明けの最初の点を失わないため |

---

## 4. DBスキーマ（確定）

`backend/internal/infrastructure/persistence/schema.sql`。起動時に毎回そのまま実行される（冪等なDDLのみ）。
**PostGISは使わない。** 空間関数も空間インデックスも必要ない。

### rooms

| 列 | 型 | 制約 | 備考 |
|---|---|---|---|
| `code` | `TEXT` | PK | 6文字の部屋コード |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default now() | |

### members

| 列 | 型 | 制約 | 備考 |
|---|---|---|---|
| `room_code` | `TEXT` | NOT NULL, FK → `rooms(code)` ON DELETE CASCADE | |
| `id` | `TEXT` | | クライアント生成のメンバーID |
| `name` | `TEXT` | NOT NULL | 1〜16文字（サーバーで切り詰め） |
| `color` | `TEXT` | NOT NULL | `#rrggbb` |
| `joined_at` | `TIMESTAMPTZ` | NOT NULL, default now() | |
| | | **PK: (`room_code`, `id`)** | |

> **主キーが複合である理由**: `id` は端末の localStorage 由来なので、同じ端末が複数の部屋に入る。
> `id` 単独をPKにすると、2つめの部屋への参加が1つめの行を上書きし、
> 再起動後の復元でメンバー情報（色・名前）が失われる。

### points

| 列 | 型 | 制約 | 備考 |
|---|---|---|---|
| `id` | `BIGSERIAL` | PK | |
| `room_code` | `TEXT` | NOT NULL | **FKを張らない**（下記） |
| `member_id` | `TEXT` | NOT NULL | **FKを張らない**（下記） |
| `lat` | `DOUBLE PRECISION` | NOT NULL | |
| `lng` | `DOUBLE PRECISION` | NOT NULL | |
| `accuracy` | `REAL` | NULL可 | 読み出し時は `NULL → 0` |
| `recorded_at` | `TIMESTAMPTZ` | NOT NULL | `Point.TS` を変換したもの |

インデックス: `points (room_code, member_id, recorded_at)` — 復元時の読み出し順そのまま。

> **`points` に外部キーを張らない理由**（重要）
>
> 1. `UpsertMember` は非同期goroutineで走るため、`members` の行が入る前に点がflushされうる。
>    FKがあると外部キー違反でflushが丸ごと失敗する。
> 2. `CopyFrom`（バルク投入）はFK検査があると遅くなる。
>
> その代わり `room_code` / `member_id` を**非正規化して持たせている**ので、
> 復元時に `members` をJOINせずに点を読める。

### 列を追加するとき

`schema.sql` は起動時に毎回流れるので、**冪等なDDLだけを追記する**。

```sql
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS title TEXT;
```

`CREATE TABLE` を書き換えるのではなく `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` を足す。
既存DBを作り直さずに済む。マイグレーションツールは入れない。

---

## 5. WebSocket プロトコル（確定）

エンドポイント: `GET /ws?room=<CODE>`

- **最初の1通は必ず `join`**。それ以外が来たら `1008 ClosePolicyViolation` で切断。
- サーバー→クライアントの `ping` は30秒ごと（Cloudflareの無通信100秒切断対策）。
  クライアントは `pong` を自動返信する（ブラウザ標準動作）。
- サーバー側の読み取り上限は **64KB/メッセージ**。

### 共通形

全メッセージは `t` フィールドで種別を判別する。

### Client → Server

| `t` | ペイロード | 備考 |
|---|---|---|
| `join` | `{ member: Member }` | 接続後1通目。必須 |
| `points` | `{ pts: Point[] }` | 1通あたり最大500点（64KB上限との兼ね合い） |
| `member` | `{ member: Member }` | 名前・色の変更。`member.id` は無視され接続時のIDが使われる |

```json
{"t":"join","member":{"id":"a3f1c8d920b74e55","name":"たろう","color":"#ff2d78"}}
{"t":"points","pts":[[43.0607,141.3544,1761278400000,8.2],[43.0608,141.3545,1761278403000,7.5]]}
{"t":"member","member":{"id":"a3f1c8d920b74e55","name":"たろう","color":"#00e5ff"}}
```

### Server → Client

| `t` | ペイロード | いつ |
|---|---|---|
| `snapshot` | `{ room, you, members, tracks }` | join直後に1回。**現在の絵をまるごと** |
| `points` | `{ member, pts }` | 他メンバーの増分。**自分の点は返ってこない** |
| `member` | `{ member }` | 誰かが参加 or 名前/色を変更 |
| `leave` | `{ member }` | 全タブが切断されたとき（ID単位） |

```json
{
  "t": "snapshot",
  "room": "KZ26AB",
  "you":     {"id":"a3f1...","name":"たろう","color":"#ff2d78"},
  "members": [{"id":"a3f1...","name":"たろう","color":"#ff2d78"}],
  "tracks":  {"a3f1...": [[43.0607,141.3544,1761278400000,8.2]]}
}
{"t":"points","member":"b7c2...","pts":[[43.0611,141.3550,1761278410000,6.1]]}
{"t":"leave","member":"b7c2..."}
```

### 不変条件

- `snapshot.tracks` のキーは必ず `snapshot.members` に含まれる…**とは限らない**。
  退出済みメンバーの軌跡はDB復元時に残るが `members` に無い場合がある
  → クライアントはフォールバック色 `#888888` を使う（`RoomView.tsx`）。
- **自分の点はサーバーからエコーバックされない。** クライアントは送信前にローカルで描画する。
  二重描画を避けるためで、ここを変えると線が重なる。
- **未知の `t` は両側で黙って無視する。** これがプロトコル拡張の唯一のルール
  （Go: `switch in.T` の default なし / TS: `switch msg.t` の default なし）。
  新しいメッセージ種別は、古いクライアントを壊さずに足せる。

### サーバー側のバリデーション（`sanitizeMember`）

クライアント申告値は信用しない。

| 項目 | 不正時の扱い |
|---|---|
| `id` が空 or 33文字以上 | サーバーが再生成 |
| `name` が空 | `"ゲスト"` |
| `name` が17文字以上 | 16文字に切り詰め（rune単位） |
| `color` が `^#[0-9a-fA-F]{6}$` に合わない | パレットから参加順で自動割り当て |

---

## 6. REST エンドポイント（確定）

| メソッド・パス | リクエスト | レスポンス |
|---|---|---|
| `POST /api/rooms` | なし | `{"code":"KZ26AB"}` |
| `GET /api/rooms/{code}` | なし | `200 {"code":"KZ26AB"}` / `404 {"error":"room not found"}` |
| `GET /api/health` | なし | `{"ok":true}` |
| `GET /ws?room=<CODE>` | — | WebSocket upgrade |
| `GET /*` | — | 埋め込みSPA（未知パスは `index.html`） |

`GET /api/rooms/{code}` は**打ち間違いを弾くためだけ**の存在確認。メンバー一覧は返さない
（それは `snapshot` の仕事）。

---

## 7. サーバー内メモリ構造（確定）

```go
Hub
 ├ rooms   map[string]*Room      // mutexで保護
 └ store   *Store                // nil ならインメモリのみ

Room                             // 全フィールドは run() の goroutine が単独所有
 ├ Code     string
 ├ cmds     chan func(*Room)     // 外部からはクロージャを送り込む（actorパターン）
 ├ closed   chan struct{}
 ├ clients  map[*Client]bool
 ├ members  map[string]Member    // memberID → Member
 ├ tracks   map[string][]Point   // memberID → 全点。これが真実のソース
 ├ pending  map[string][]Point   // DB未書き込み分。flushでクリア
 └ seq      int                  // 色の自動割り当て用カウンタ

Client
 ├ room    *Room
 ├ conn    *websocket.Conn
 ├ out     chan []byte           // 容量64。溢れたら切断
 ├ member  Member
 └ dead    bool                  // Room goroutine からのみ触る
```

### 守るべき不変条件

1. **`Room` のフィールドは `run()` goroutine 以外から触らない。** 外からは `r.do(func(r *Room){...})`。
2. `tracks` と `pending` は同じ点を持つ。`pending` だけが flush でクリアされる。
3. 同一 `member.ID` の `Client` が複数ありうる（複数タブ）。`leave` は最後の1つが切れたときだけ送る。
4. DB書き込み（`SavePoints` / `UpsertMember`）は goroutine に逃がす。`run()` をブロックさせない。

---

## 8. クライアント状態（確定）

### zustand ストア（`frontend/src/store.ts`）

```ts
{
  me:           Member                             // localStorage 永続
  room:         string | null
  status:       "idle" | "connecting" | "open" | "closed"
  members:      Record<string, Member & { online: boolean }>
  stats:        Record<string, { points: number; meters: number }>
  recording:    boolean
  mock:         boolean
  wakeLockHeld: boolean
  error:        string | null
}
```

### 軌跡は zustand に入れない（重要）

点列は `useTracks` の `useRef` が持つ。**Reactの状態にしない。**
毎秒頂点が増えるため、状態に載せると毎秒全体が再レンダリングされる。

```ts
Map<memberId, {
  group:    L.LayerGroup   // 地図に載っているレイヤ
  segments: L.Polyline[]   // ギャップで分割された区間。線を繋げないため
  points:   Point[]        // 統計・書き出し用
  color:    string
}>
```

- `segments` が複数になるのは `GAP_MS`（30秒）以上空いた箇所。
  バックグラウンドで測位が止まった区間を直線で繋がない。
- 再レンダリングが走るのは `stats` 更新時だけ（3秒ごとのflush単位 + 自分の点ごと）。

---

## 9. 永続キー

| 場所 | キー | 値 | 寿命 |
|---|---|---|---|
| localStorage | `gpsart:me` | `Member`（JSON） | 永続。消すと別人になる |
| IndexedDB (idb-keyval) | `outbox:<ROOMCODE>` | `Point[]` | 送信成功で削除。上限20,000点（超過分は古い方を捨てる） |
| URL クエリ | `?room=<CODE>` | 部屋コード | QR・共有リンクからの直接参加 |

`outbox` はWS切断中の点を保持する。再接続時に500点ずつ送り直し、
途中で失敗したら残りを書き戻す（`frontend/src/lib/ws.ts` の `flushOutbox`）。

---

## 10. 定数一覧（連動しているものに注意）

| 定数 | 値 | 場所 | 連動相手 |
|---|---|---|---|
| `MAX_ACCURACY_M` | 30 | `frontend/src/lib/geo.ts` | — |
| `MIN_DISTANCE_M` | 5 | 同 | — |
| `MAX_SPEED_MPS` | 14 | 同 | — |
| `GAP_MS` | 30,000 | 同 | `useTracks` のセグメント分割、`pathLength` の距離除外 |
| `FLUSH_MS` | 3,000 | `frontend/src/hooks/useRecorder.ts` | 送信頻度。`stats` の更新頻度でもある |
| `CHUNK` | 500 | `frontend/src/lib/ws.ts` | **`maxMessageSize` 64KB を超えない上限** |
| `MAX`（outbox） | 20,000 | `frontend/src/lib/outbox.ts` | — |
| `pingPeriod` | 30s | `internal/interface/ws/client.go` | Cloudflareの無通信100秒切断。**これより短く保つ** |
| `pongWait` | 70s | 同 | `pingPeriod` の2倍強にしておく |
| `maxMessageSize` | 65,536 | 同 | **`CHUNK` と連動。片方だけ変えない** |
| `outBuffer` | 64 | 同 | 溢れると遅いクライアントを切断する閾値 |
| `flushInterval` | 5s | `internal/usecase/session.go` | DB書き込み頻度 |

---

## 11. 拡張の作法（後から決まるもの）

**以下はコードに入っていない。** 既存構造を壊さずに足すための方針だけ決めてある。

### 共通ルール

| 対象 | やり方 |
|---|---|
| 新しいWSメッセージ | 新しい `t` を足すだけ。未知の `t` は両側で無視されるので古いクライアントは壊れない |
| DBの列追加 | `schema.sql` に `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` を**追記**する |
| 形が未確定なもの | `JSONB` 列1本で持つ。スキーマ変更なしに中身を変えられる |
| 点から導出できるもの | **保存しない**。クライアントで計算する |

### リプレイのタイムラプス

**追加データ構造は不要。** `snapshot.tracks` に `ts` が入っているので、
クライアントが時刻でソートして再生するだけ。

点数が多い部屋で初期ロードが重くなったら、WSの `snapshot` とは別に
`GET /api/rooms/{code}/tracks` を足して `simplify()` 済みの軌跡を返す
（`frontend/src/lib/geo.ts` に Douglas-Peucker を用意済み）。

### お題レイヤー（目標の文字・図形）

形が決まっていないので `JSONB` で持つのが安全。

```sql
ALTER TABLE rooms ADD COLUMN IF NOT EXISTS goal JSONB;
```

想定する中身（**未確定。決まったら差し替える**）:

```json
{
  "kind": "polyline",
  "title": "大通公園に『JP』",
  "strokes": [[[43.0607,141.3544],[43.0611,141.3550]], [[43.0615,141.3560]]],
  "assignments": { "a3f1c8d920b74e55": [0, 2] },
  "toleranceM": 20
}
```

達成率は**クライアントで計算する**（各 stroke から `toleranceM` 以内を通過した割合）。
サーバーに計算させない。

WSには `{"t":"goal","goal":{...}}` を足す。

### 塗りつぶしグリッド

**永続化しない。** 点から導出する。セルIDの決め方だけ先に固定しておく:

```ts
const CELL_DEG = 0.00045;  // ≒50m（緯度方向）
const cellId = (p: Point) =>
  `${Math.floor(p[0] / CELL_DEG)}:${Math.floor(p[1] / CELL_DEG)}`;
```

固定グリッドなので**全員が同じ点から同じセルIDを出す**。これが守られていれば
クライアント集計とサーバー集計が一致するので、どちらで計算してもよくなる。
経度方向の実距離は緯度で変わるが、スコアリング用途なら許容する。

### 画像・GeoJSON書き出し

`useTracks` の `snapshot()` が `Record<memberId, Point[]>` を返すので、そこから作る。

```ts
// GeoJSON は [lng, lat] 順。Point タプルと逆なので必ず入れ替える
{
  type: "FeatureCollection",
  features: [{
    type: "Feature",
    geometry: { type: "LineString", coordinates: pts.map(p => [p[1], p[0]]) },
    properties: { memberId, name, color }
  }]
}
```

ギャップで線を切るなら `splitByGap()`（実装済み）で区間に分けて `MultiLineString` にする。

### まだ決めていないこと

| 項目 | メモ |
|---|---|
| 部屋のタイトル | `rooms.title TEXT` を足すだけ。未着手 |
| 部屋の有効期限・削除 | 今は無期限。`rooms` を消せば `members` はCASCADEで消えるが **`points` は消えない**（FKが無いため）。消すなら `DELETE FROM points WHERE room_code = $1` を明示的に書く |
| メンバーのオンライン状態の永続 | 今はメモリのみ（`online`）。再起動すると全員オフライン扱い |
| 記録セッションの区切り | 今は「記録開始/終了」がDBに残らない。必要なら `points` に `session_id` を足す |
| 認証 | 作らない。`member.id` は自己申告 |
