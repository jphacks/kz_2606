# フロントエンド（3人）

前提: [DATA_MODEL.md](./DATA_MODEL.md) 3章・5章・6章・9章 / [TECH_PLAN.md](./TECH_PLAN.md) 7章

## 担当割り — 1ファイル1人。同じファイルを2人で触らない

| | 担当 | 持ちファイル |
|---|---|---|
| **F1** | 地図・軌跡（一番むずかしい） | `hooks/useLeafletMap.ts` `hooks/useTracks.ts` `lib/geo.ts` |
| **F2** | 測位・通信・状態 | `lib/types.ts` `lib/gps.ts` `lib/ws.ts` `lib/outbox.ts` `lib/api.ts` `hooks/useRecorder.ts` `store.ts` |
| **F3** | 画面 | `App.tsx` `components/*` `index.css` |

## 最初にやること（F2が10分で、他2人はこれを待つ）

**`lib/types.ts` をpushする。** DATA_MODEL.md 3章・6章をそのまま写すだけ。

```ts
export type Point = [lat: number, lng: number, ts: number, acc: number];
export type Member = { id: number; displayName: string; color: string;
                       online: boolean; banned: boolean; recording: boolean };
export type Route  = { id: number; member: number; color: string;
                       endedAt: string | null; pts: Point[] };
export type ServerMsg = ... // snapshot | route | points | route_end | member | leave
                            // | closed | reopened | banned | unbanned | route_deleted | error
```

**IDはすべて `number`**（サーバーが `BIGSERIAL` で発行する）。クライアントは生成しません。

次に `store.ts` の形だけpush（中身は空でいい）。これで3人が独立して動けます。

## 時間の流れ — バックを待たずに進める

| 時間 | F1 | F2 | F3 |
|---|---|---|---|
| 0–1h | 地図を出すだけ | `types.ts` → `store.ts` | ログイン/登録画面のガワ |
| 1–3h | **手書きの点列で `useTracks` を完成**（WS不要） | `gps.ts` → `useRecorder.ts`（ローカル描画まで） | `JoinScreen` → `Controls` |
| 3h– | `geo.ts` のフィルタを詰める | **バックの `/ws` ができたら `ws.ts` を繋ぐ** | `RoomView` で結線 |
| 後半 | セグメント分割の調整 | 認証API（`/me` 含む） | owner用UI（締める・BAN） |

**待ち合わせはここ1点だけ**: F2 の `ws.ts` ⇔ バックの `/ws`。それ以外は並行で進みます。

## F1 の手順

1. `useLeafletMap` — 地図を1個だけ作って ref で持つ。CARTO `dark_all`、**`preferCanvas: true`**
2. `useTracks` — **`Map<routeId, {...}>`**。route 1本 = ポリラインの束
3. **3段階でセグメントを分ける**
   - route が変わったら新しいレイヤグループ
   - 同じ route 内で `GAP_MS`（30秒）以上空いたら新しい `L.Polyline`
   - **1000点ごとにも切る**（Leaflet が画面外のレイヤをスキップできるようになる）
4. **`setLatLngs` と `addLatLng` を使い分ける**
   - snapshot の一括投入 → `setLatLngs()`（1回しか再描画しない）
   - 増分 → `addLatLng()`
   - snapshot を `addLatLng` のループで入れると固まります
5. `geo.ts` — `distance`（haversine）/ `PointFilter` / `simplify`
6. **累積距離を持つ。** `pathLength` を毎回全点に対して呼ばない

**ゴール**: 適当な点列を `append()` に流したら線が出る。ギャップで切れる。色を変えると全セグメント変わる

## F2 の手順

1. `types.ts` → `store.ts`（zustand）
2. `gps.ts` — `watchReal` と `watchMock` を同じ形で。**モックを先に作る**（室内開発のため）
3. `useRecorder` — 測位 → `PointFilter` → **即時描画** / 送信は3秒ごとにまとめる
4. `ws.ts` — 指数バックオフ再接続。**`start` / `points` / `stop` を送る**
5. `outbox.ts` — 切断中の点を IndexedDB に積む。**`routeId` と一緒に積むこと**
6. `api.ts` — auth（register / login / logout / **me**）、rooms（create / get / join）

**`GET /api/auth/me` をアプリ起動時に必ず叩く。** これが無いとリロードでログイン状態が消えます。

**ゴール**: モックGPSで記録開始すると点が流れ、WSを落としても溜まって再送される

## F3 の手順

1. `LoginScreen` — 登録 / ログイン。**登録成功でそのままログイン済みになる**
2. `RoomListOrJoin` — 部屋を作る（title入力） / コードで参加（displayName・色）
3. `Controls` — 記録開始/終了、接続状態、**モックGPS切替**、色選択
4. `MemberList` — 色・名前・距離・**記録中インジケータ**・BAN済みの表示
5. `ShareDialog` — 部屋コード + QR（`qrcode`）
6. `OwnerPanel` — **owner だけに出す**: 部屋を締める / 再開 / メンバーをBAN
7. `App.tsx` / `RoomView` — `?room=CODE` で直接参加、地図とUIの重ね合わせ

**ゴール**: 登録 → 部屋作成 → QR → 別端末で参加 → 記録 → 締める が一周する

## 落とし穴

- **点列をReactの状態（zustand/useState）に入れない。** 毎秒再レンダリングになります。
  `useRef` + 命令的に `addLatLng`
- **react-leaflet を入れない。** 同じ理由
- **`PointFilter` を飛ばさない。** 生データのまま描くと団子とヒゲだらけになります
- **自分の点はサーバーから返ってこない。** 送信前にローカルで描く（待つと体感が重い）
- **再送中は直送を止める。** `outbox` の吐き出しは非同期なので、その隙間に新しい点を直送すると
  **時刻が逆行した点列**が他クライアントに届きます。`draining` フラグを立ててください
- **`banned: true` のメンバーの route は非表示にする**（DBからは消えていない）
- **`leave` と `route_end` を区別する。** 前者は切断、後者は記録終了（まだ部屋にいる）
- **iOSは画面ロックで測位が止まる。** WakeLockを入れて、UIも「手に持って歩く」前提で作る
- モバイル対応: `100dvh`、`env(safe-area-inset-*)`、`touch-action` / `overscroll-behavior`

## やらないこと（最初は）

リプレイのタイムラプス / お題レイヤー / 画像書き出し。
一周動いてから、`useTracks.snapshot()` と `geo.simplify()` の上に載せます
