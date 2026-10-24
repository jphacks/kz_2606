package domain

import (
	"math"
	"testing"
	"time"
)

func newStatsFixture() RoomSnapshot {
	bannedAt := time.Date(2026, 10, 24, 4, 0, 0, 0, time.UTC)
	return RoomSnapshot{
		Room: Room{
			ID:        7,
			Code:      "KZ26AB",
			CreatedAt: time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC),
		},
		Members: []Member{
			{ID: 1, DisplayName: "はなこ"},
			{ID: 3, DisplayName: "たろう"},
			{ID: 5, DisplayName: "じろう", BannedAt: &bannedAt},
		},
		Routes: []Route{
			{ID: 12, MemberID: 1, Points: []Point{
				{Lat: 43.0607, Lng: 141.3544, TS: 1000},
				{Lat: 43.0611, Lng: 141.3550, TS: 2000},
			}},
			{ID: 13, MemberID: 3, Points: []Point{
				{Lat: 43.0611, Lng: 141.3550, TS: 1000},
				{Lat: 43.0615, Lng: 141.3556, TS: 2000},
			}},
			// BAN されたメンバーの route。フロントは非表示にするので合計にも入れない
			{ID: 14, MemberID: 5, Points: []Point{
				{Lat: 43.0607, Lng: 141.3544, TS: 1000},
				{Lat: 43.0700, Lng: 141.3700, TS: 2000},
			}},
		},
	}
}

func TestStatsSumsDistanceExcludingBannedMembers(t *testing.T) {
	stats := newStatsFixture().Stats(time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC))
	if math.Abs(stats.Meters-131.975435) > 0.001 {
		t.Fatalf("Meters = %f, want 131.975435（BAN メンバーの route を除いた2本の合計）", stats.Meters)
	}
}

func TestStatsCountsOnlyActiveMembers(t *testing.T) {
	stats := newStatsFixture().Stats(time.Now())
	if stats.Members != 2 {
		t.Fatalf("Members = %d, want 2（BAN を除く）", stats.Members)
	}
}

func TestStatsElapsedUsesNowWhileRoomIsOpen(t *testing.T) {
	snapshot := newStatsFixture()
	stats := snapshot.Stats(time.Date(2026, 10, 24, 6, 30, 0, 0, time.UTC))
	if stats.Elapsed != 90*time.Minute {
		t.Fatalf("Elapsed = %v, want 1h30m", stats.Elapsed)
	}
	if !stats.StartedAt.Equal(snapshot.Room.CreatedAt) {
		t.Fatalf("StartedAt = %v, want %v", stats.StartedAt, snapshot.Room.CreatedAt)
	}
}

func TestStatsElapsedStopsAtClosedAt(t *testing.T) {
	snapshot := newStatsFixture()
	closedAt := time.Date(2026, 10, 24, 5, 45, 0, 0, time.UTC)
	snapshot.Room.ClosedAt = &closedAt

	// 締めたあとに時間が経っても、経過時間は締めた時点で止まる
	stats := snapshot.Stats(time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC))
	if stats.Elapsed != 45*time.Minute {
		t.Fatalf("Elapsed = %v, want 45m（closedAt で止まるべき）", stats.Elapsed)
	}
}

func TestStatsElapsedNeverGoesNegative(t *testing.T) {
	snapshot := newStatsFixture()
	stats := snapshot.Stats(snapshot.Room.CreatedAt.Add(-time.Hour))
	if stats.Elapsed != 0 {
		t.Fatalf("Elapsed = %v, want 0", stats.Elapsed)
	}
}
