-- PostGIS は不要。空間関数も空間インデックスも使わない。
-- 起動時に毎回そのまま実行されるので、冪等なDDLだけを書く。
-- 列を足すときは ALTER TABLE ... ADD COLUMN IF NOT EXISTS を追記する。
-- 仕様: DATA_MODEL.md 4章

CREATE TABLE IF NOT EXISTS rooms (
    code       TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 主キーは (room_code, id)。member.id は端末の localStorage 由来なので、
-- 同じ端末が複数の部屋に入ることがある。
CREATE TABLE IF NOT EXISTS members (
    room_code TEXT NOT NULL REFERENCES rooms (code) ON DELETE CASCADE,
    id        TEXT NOT NULL,
    name      TEXT NOT NULL,
    color     TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (room_code, id)
);

-- 外部キーを張らない: メンバーの保存が非同期なので points が先に来る。
-- CopyFrom も FK 検査があると遅くなる。
CREATE TABLE IF NOT EXISTS points (
    id          BIGSERIAL PRIMARY KEY,
    room_code   TEXT NOT NULL,
    member_id   TEXT NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lng         DOUBLE PRECISION NOT NULL,
    accuracy    REAL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS points_room_member_time
    ON points (room_code, member_id, recorded_at);
