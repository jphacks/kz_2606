package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

func newStatsSnapshot() (domain.RoomSnapshot, domain.Member) {
	endedAt := time.Date(2026, 10, 24, 5, 45, 0, 0, time.UTC)
	snapshot := domain.RoomSnapshot{
		Room: domain.Room{
			ID:            7,
			Code:          "KZ26AB",
			Title:         "大通公園に絵を描く",
			OwnerMemberID: 1,
		},
		Members: []domain.Member{
			{ID: 1, DisplayName: "はなこ", Color: "#00e5ff"},
			{ID: 3, DisplayName: "たろう", Color: "#ff2d78"},
		},
		Routes: []domain.Route{
			// 記録を止めた route。経過時間は endedAt で固定される
			{ID: 12, MemberID: 1, StartedAt: time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC), EndedAt: &endedAt,
				Points: []domain.Point{
					{Lat: 43.0607, Lng: 141.3544, TS: 1000},
					{Lat: 43.0611, Lng: 141.3550, TS: 2000},
				}},
			// 記録中の route。経過時間は now まで伸びる
			{ID: 14, MemberID: 3, StartedAt: time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC),
				Points: []domain.Point{}},
		},
	}
	return snapshot, domain.Member{ID: 3, DisplayName: "たろう", Color: "#ff2d78"}
}

// route ごとの集計のワイヤ形式（キー名）を生JSONで確認する。
// 構造体にデコードすると json タグの変更を検出できないため。
func TestSnapshotRouteStatsWireKeys(t *testing.T) {
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
	if _, ok := raw["stats"]; ok {
		t.Fatalf("room 単位の stats は廃止したはずです: %s", payload)
	}
	routes, ok := raw["routes"].([]any)
	if !ok || len(routes) != 2 {
		t.Fatalf("routes = %v", raw["routes"])
	}
	route, ok := routes[0].(map[string]any)
	if !ok {
		t.Fatalf("route の形が想定外: %v", routes[0])
	}
	for _, key := range []string{"id", "member", "color", "startedAt", "endedAt", "meters", "elapsedMs", "pts"} {
		if _, ok := route[key]; !ok {
			t.Fatalf("routes[].%s がありません: %v", key, route)
		}
	}
}

func TestSnapshotRouteStatsValues(t *testing.T) {
	snapshot, you := newStatsSnapshot()
	now := time.Date(2026, 10, 24, 6, 30, 0, 0, time.UTC)

	dto := makeSnapshotDTO(snapshot, you, NewRegistry(), now)
	if len(dto.Routes) != 2 {
		t.Fatalf("routes = %d 本", len(dto.Routes))
	}

	// 止めた route: 05:00 → 05:45 の45分で固定、距離は2点分
	ended := dto.Routes[0]
	if ended.ElapsedMs != 45*60*1000 {
		t.Errorf("止めた route の elapsedMs = %d, want %d", ended.ElapsedMs, 45*60*1000)
	}
	if ended.Meters < 65.9 || ended.Meters > 66.1 {
		t.Errorf("止めた route の meters = %f, want 約65.99", ended.Meters)
	}

	// 記録中の route: 06:00 → now(06:30) の30分、点が無いので距離0
	recording := dto.Routes[1]
	if recording.ElapsedMs != 30*60*1000 {
		t.Errorf("記録中 route の elapsedMs = %d, want %d", recording.ElapsedMs, 30*60*1000)
	}
	if recording.Meters != 0 {
		t.Errorf("記録中 route の meters = %f, want 0", recording.Meters)
	}
	if !recording.StartedAt.Equal(snapshot.Routes[1].StartedAt) {
		t.Errorf("startedAt = %v, want %v", recording.StartedAt, snapshot.Routes[1].StartedAt)
	}
}
