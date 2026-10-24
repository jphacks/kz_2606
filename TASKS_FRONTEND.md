# フロントエンド（3人）

前提: [DATA_MODEL.md](./DATA_MODEL.md) 3章・5章・8章 / [TECH_PLAN.md](./TECH_PLAN.md) 7章

## 担当割り — 1ファイル1人。同じファイルを2人で触らない

| | 担当 | 持ちファイル |
|---|---|---|
| **F1** | 地図・軌跡（一番むずかしい） | `hooks/useLeafletMap.ts` `hooks/useTracks.ts` `lib/geo.ts` |
| **F2** | 測位・通信・状態 | `lib/types.ts` `lib/gps.ts` `lib/ws.ts` `lib/outbox.ts` `lib/api.ts` `hooks/useRecorder.ts` `store.ts` |
| **F3** | 画面 | `App.tsx` `components/*` `index.css` |

## 最初にやること（F2が10分で、他2人はこれを待つ）

**`lib/types.ts` をpushする。** DATA_MODEL.md 3章・5章をそのまま写すだけ。

```ts
export type Point = [lat: number, lng: number, ts: number, acc: number];
export type Member = { id: string; name: string; color: string };
export type ServerMsg = ... // snapshot | points | member | leave
```

次に `store.ts` の形だけpush（中身は空でいい）。これで3人が独立して動ける。

## 時間の流れ — バックを待たずに進める

| 時間 | F1 | F2 | F3 |
|---|---|---|---|
| 0–1h | 地図を出すだけ | `types.ts` → `store.ts` | 画面のガワだけ（ダミーデータ） |
| 1–3h | **手書きの点列で `useTracks` を完成**（WS不要） | `gps.ts` → `useRecorder.ts`（ローカル描画まで） | `JoinScreen` → `Controls` 完成 |
| 3h– | `geo.ts` のフィルタを詰める | **バックの `/ws` ができたら `ws.ts` を繋ぐ** | `RoomView` で全部を結線 |

**待ち合わせはここ1点だけ**: F2 の `ws.ts` ⇔ バックの `/ws`。それ以外は並行で進む。

## F1 の手順

1. `useLeafletMap` — Leafletの地図を1個だけ作って ref で持つ。タイルは CARTO `dark_all`、`preferCanvas: true`
2. `useTracks` — `Map<memberId, {group, segments, points, color}>` を命令的に伸ばす
3. **30秒以上空いたら新しい `L.Polyline` を作る**（線を繋げない）
4. `geo.ts` — `distance`（haversine）/ `pathLength` / `PointFilter`

**ゴール**: 適当な点列を `append()` に流したら線が出る。ギャップで切れる。色を変えると全セグメント変わる

## F2 の手順

1. `types.ts` → `store.ts`（zustand）
2. `gps.ts` — `watchReal` と `watchMock` を同じ形で。**モックを先に作る**（室内で開発するため）
3. `useRecorder` — 測位 → `PointFilter` → **即時描画** / 送信は3秒ごとにまとめる
4. `ws.ts` — 指数バックオフ再接続。`onopen` で `join` を送る
5. `outbox.ts` — 切断中の点を IndexedDB に積んで再接続時に500点ずつ再送
6. `api.ts` — `createRoom` / `roomExists`

**ゴール**: モックGPSで記録開始すると点が流れ、WSを落としても溜まって再送される

## F3 の手順

1. `JoinScreen` — 名前・色（固定パレット）・部屋を作る / コードで参加
2. `Controls` — 記録開始/終了、接続状態、**モックGPS切替**、色選択
3. `MemberList` — 色・名前・距離
4. `ShareDialog` — 部屋コード + QR（`qrcode`）
5. `App.tsx` / `RoomView` — `?room=CODE` で直接参加、地図とUIの重ね合わせ

**ゴール**: 部屋作成 → QR → 別端末で参加 → 記録ボタン が一周する

## 落とし穴

- **点列をReactの状態（zustand/useState）に入れない。** 毎秒再レンダリングになる。`useRef` + 命令的に `addLatLng`
- **react-leaflet を入れない。** 同じ理由
- **`PointFilter` を飛ばさない。** 生データのまま描くと団子とヒゲだらけになる
- **自分の点はサーバーから返ってこない。** 送信前にローカルで描く（待つと体感が重い）
- **iOSは画面ロックで測位が止まる。** WakeLockを入れて、UIも「手に持って歩く」前提で作る
- モバイル対応: `100dvh`、`env(safe-area-inset-*)`、`touch-action` / `overscroll-behavior`

## やらないこと（最初は）

リプレイのタイムラプス / お題レイヤー / 画像書き出し。
一周動いてから、`useTracks.snapshot()` と `geo.simplify()` の上に載せる
