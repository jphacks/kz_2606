-- PostGIS は不要。空間関数も空間インデックスも使わない。
-- 起動時に毎回そのまま実行されるので、冪等なDDLだけを書く。
-- 列を足すときは ALTER TABLE ... ADD COLUMN IF NOT EXISTS を追記する。
-- 仕様: DATA_MODEL.md 4章

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,   -- 小文字で保存
    password_hash TEXT NOT NULL,          -- bcrypt
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS rooms (
    id            BIGSERIAL PRIMARY KEY,
    code          TEXT NOT NULL UNIQUE,   -- 6文字。URL/QRで共有するのはこれ（id は外に出さない）
    title         TEXT NOT NULL,
    owner_user_id BIGINT NOT NULL REFERENCES users (id),
    closed_at     TIMESTAMPTZ,            -- NULL = 進行中
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS members (
    id           BIGSERIAL PRIMARY KEY,
    room_id      BIGINT NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    user_id      BIGINT NOT NULL REFERENCES users (id),
    display_name TEXT NOT NULL,
    color        TEXT NOT NULL,           -- #rrggbb
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    banned_at    TIMESTAMPTZ,             -- NULL = 参加中
    UNIQUE (room_id, user_id)             -- 再入室・リロードが冪等になる
);

-- routes = 1メンバーの1回の記録。記録開始で作り、終了で ended_at を埋める。
-- 記録開始時に同期で作るので points より必ず先に存在する → 外部キーを張れる。
CREATE TABLE IF NOT EXISTS routes (
    id         BIGSERIAL PRIMARY KEY,
    room_id    BIGINT NOT NULL REFERENCES rooms (id) ON DELETE CASCADE,
    member_id  BIGINT NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at   TIMESTAMPTZ              -- NULL = 記録中
);

CREATE INDEX IF NOT EXISTS routes_room ON routes (room_id);

-- points は route_id だけで辿れる（member_id は routes から引ける）
CREATE TABLE IF NOT EXISTS points (
    id          BIGSERIAL PRIMARY KEY,
    route_id    BIGINT NOT NULL REFERENCES routes (id) ON DELETE CASCADE,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    accuracy    REAL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS points_route_time ON points (route_id, recorded_at);
