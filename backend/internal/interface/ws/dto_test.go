package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

func newStatsSnapshot() (domain.RoomSnapshot, domain.Member) {
	bannedAt := time.Date(2026, 10, 24, 4, 0, 0, 0, time.UTC)
	snapshot := domain.RoomSnapshot{
		Room: domain.Room{
			ID:            7,
			Code:          "KZ26AB",
			Title:         "大通公園に絵を描く",
			OwnerMemberID: 1,
			CreatedAt:     time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC),
		},
		Members: []domain.Member{
			{ID: 1, DisplayName: "はなこ", Color: "#00e5ff"},
			{ID: 3, DisplayName: "たろう", Color: "#ff2d78"},
			{ID: 5, DisplayName: "じろう", Color: "#ffe600", BannedAt: &bannedAt},
		},
		Routes: []domain.Route{
			{ID: 12, MemberID: 1, Points: []domain.Point{
				{Lat: 43.0607, Lng: 141.3544, TS: 1000},
				{Lat: 43.0611, Lng: 141.3550, TS: 2000},
			}},
		},
	}
	return snapshot, domain.Member{ID: 3, DisplayName: "たろう", Color: "#ff2d78"}
}

// stats のワイヤ形式（キー名）を生JSONで確認する。
// 構造体にデコードすると json タグの変更を検出できないため。
func TestSnapshotStatsWireKeys(t *testing.T) {
	snapshot, you := newStatsSnapshot()
	now := time.Date(2026, 10, 24, 6, 30, 0, 0, time.UTC)

	payload, err := json.Marshal(makeSnapshotDTO(snapshot, you, NewRegistry(), now))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	stats, ok := raw["stats"].(map[string]any)
	if !ok {
		t.Fatalf("snapshot に stats がありません: %s", payload)
	}
	for _, key := range []string{"meters", "members", "startedAt", "elapsedMs"} {
		if _, ok := stats[key]; !ok {
			t.Fatalf("stats.%s がありません: %v", key, stats)
		}
	}
}

func TestSnapshotStatsValues(t *testing.T) {
	snapshot, you := newStatsSnapshot()
	now := time.Date(2026, 10, 24, 6, 30, 0, 0, time.UTC)

	dto := makeSnapshotDTO(snapshot, you, NewRegistry(), now)
	if dto.Stats.Members != 2 {
		t.Errorf("stats.members = %d, want 2（BAN を除く）", dto.Stats.Members)
	}
	if dto.Stats.ElapsedMs != 90*60*1000 {
		t.Errorf("stats.elapsedMs = %d, want %d", dto.Stats.ElapsedMs, 90*60*1000)
	}
	if !dto.Stats.StartedAt.Equal(snapshot.Room.CreatedAt) {
		t.Errorf("stats.startedAt = %v, want %v", dto.Stats.StartedAt, snapshot.Room.CreatedAt)
	}
	if dto.Stats.Meters < 65.9 || dto.Stats.Meters > 66.1 {
		t.Errorf("stats.meters = %f, want 約65.99", dto.Stats.Meters)
	}
}

// 締めたあとは経過時間が止まることをワイヤ形式でも確認する
func TestSnapshotStatsElapsedStopsWhenClosed(t *testing.T) {
	snapshot, you := newStatsSnapshot()
	closedAt := time.Date(2026, 10, 24, 5, 45, 0, 0, time.UTC)
	snapshot.Room.ClosedAt = &closedAt

	dto := makeSnapshotDTO(snapshot, you, NewRegistry(), time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC))
	if dto.Stats.ElapsedMs != 45*60*1000 {
		t.Fatalf("stats.elapsedMs = %d, want %d", dto.Stats.ElapsedMs, 45*60*1000)
	}
}
