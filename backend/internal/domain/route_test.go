package domain

import (
	"math"
	"testing"
	"time"
)

func newRouteFixture() Route {
	return Route{
		ID:        12,
		MemberID:  1,
		StartedAt: time.Date(2026, 10, 24, 5, 0, 0, 0, time.UTC),
		Points: []Point{
			{Lat: 43.0607, Lng: 141.3544, TS: 1000},
			{Lat: 43.0611, Lng: 141.3550, TS: 2000},
			{Lat: 43.0615, Lng: 141.3556, TS: 3000},
		},
	}
}

func TestRouteStatsSumsItsOwnPoints(t *testing.T) {
	stats := newRouteFixture().Stats(time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC))
	if math.Abs(stats.Meters-131.975435) > 0.001 {
		t.Fatalf("Meters = %f, want 131.975435", stats.Meters)
	}
}

func TestRouteStatsSkipsGapsOverThreshold(t *testing.T) {
	route := newRouteFixture()
	route.Points[2].TS = route.Points[1].TS + GapMillis + 1
	stats := route.Stats(time.Now())
	if math.Abs(stats.Meters-65.987835) > 0.001 {
		t.Fatalf("ギャップ区間を加算しています: Meters = %f, want 65.987835", stats.Meters)
	}
}

func TestRouteStatsCountsExactlyGapMillis(t *testing.T) {
	// 仕様 3章は「直前の点から `> 30秒`」をギャップとするので、
	// ちょうど GapMillis は**数える**。フロントの geo.ts も同じ境界。
	route := newRouteFixture()
	route.Points = route.Points[:2]
	route.Points[1].TS = route.Points[0].TS + GapMillis
	stats := route.Stats(time.Now())
	if math.Abs(stats.Meters-65.987835) > 0.001 {
		t.Fatalf("ちょうど30秒を除外しています: Meters = %f, want 65.987835", stats.Meters)
	}

	// 1ミリ秒でも超えたら数えない
	route.Points[1].TS = route.Points[0].TS + GapMillis + 1
	if stats := route.Stats(time.Now()); stats.Meters != 0 {
		t.Fatalf("30秒超を加算しています: Meters = %f, want 0", stats.Meters)
	}
}

func TestRouteStatsElapsedUsesNowWhileRecording(t *testing.T) {
	route := newRouteFixture()
	if !route.Recording() {
		t.Fatal("EndedAt が nil のとき Recording() は true のはず")
	}
	stats := route.Stats(time.Date(2026, 10, 24, 6, 30, 0, 0, time.UTC))
	if stats.Elapsed != 90*time.Minute {
		t.Fatalf("Elapsed = %v, want 1h30m", stats.Elapsed)
	}
}

func TestRouteStatsElapsedStopsAtEndedAt(t *testing.T) {
	route := newRouteFixture()
	endedAt := time.Date(2026, 10, 24, 5, 45, 0, 0, time.UTC)
	route.EndedAt = &endedAt
	if route.Recording() {
		t.Fatal("EndedAt があるとき Recording() は false のはず")
	}

	// 記録を止めたあとに時間が経っても、経過時間は止めた時点で固定される
	stats := route.Stats(time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC))
	if stats.Elapsed != 45*time.Minute {
		t.Fatalf("Elapsed = %v, want 45m（endedAt で止まるべき）", stats.Elapsed)
	}
}

func TestRouteStatsElapsedNeverGoesNegative(t *testing.T) {
	route := newRouteFixture()
	stats := route.Stats(route.StartedAt.Add(-time.Hour))
	if stats.Elapsed != 0 {
		t.Fatalf("Elapsed = %v, want 0", stats.Elapsed)
	}
}

func TestRouteStatsHandlesEmptyPoints(t *testing.T) {
	route := newRouteFixture()
	route.Points = nil
	if stats := route.Stats(time.Now()); stats.Meters != 0 {
		t.Fatalf("Meters = %f, want 0", stats.Meters)
	}
}
