package domain

import (
	"math"
	"testing"
)

func TestDistanceMatchesHaversine(t *testing.T) {
	cases := []struct {
		name     string
		from, to Point
		want     float64
	}{
		{"緯度1度", Point{Lat: 0, Lng: 0}, Point{Lat: 1, Lng: 0}, 111194.926645},
		{"札幌の近接2点", Point{Lat: 43.0607, Lng: 141.3544}, Point{Lat: 43.0611, Lng: 141.3550}, 65.987835},
		{"同じ点", Point{Lat: 43.0607, Lng: 141.3544}, Point{Lat: 43.0607, Lng: 141.3544}, 0},
	}
	for _, c := range cases {
		if got := Distance(c.from, c.to); math.Abs(got-c.want) > 0.001 {
			t.Errorf("%s: Distance = %f, want %f", c.name, got, c.want)
		}
	}
}

func TestDistanceIsSymmetric(t *testing.T) {
	from := Point{Lat: 43.0607, Lng: 141.3544}
	to := Point{Lat: 35.6812, Lng: 139.7671}
	if math.Abs(Distance(from, to)-Distance(to, from)) > 0.000001 {
		t.Fatalf("Distance が対称ではありません: %f / %f", Distance(from, to), Distance(to, from))
	}
}

func TestPathLengthSumsConsecutivePoints(t *testing.T) {
	points := []Point{
		{Lat: 43.0607, Lng: 141.3544, TS: 1000},
		{Lat: 43.0611, Lng: 141.3550, TS: 2000},
		{Lat: 43.0615, Lng: 141.3556, TS: 3000},
	}
	if got := PathLength(points); math.Abs(got-131.975435) > 0.001 {
		t.Fatalf("PathLength = %f, want 131.975435", got)
	}
}

func TestPathLengthSkipsGapsOverThreshold(t *testing.T) {
	// 2点目と3点目の間が GapMillis を超えて空いているので、その区間は加算しない
	points := []Point{
		{Lat: 43.0607, Lng: 141.3544, TS: 1000},
		{Lat: 43.0611, Lng: 141.3550, TS: 2000},
		{Lat: 43.0615, Lng: 141.3556, TS: 2000 + GapMillis + 1},
	}
	if got := PathLength(points); math.Abs(got-65.987835) > 0.001 {
		t.Fatalf("ギャップ区間を加算しています: PathLength = %f, want 65.987835", got)
	}
}

func TestPathLengthHandlesShortSlices(t *testing.T) {
	if got := PathLength(nil); got != 0 {
		t.Errorf("PathLength(nil) = %f, want 0", got)
	}
	if got := PathLength([]Point{{Lat: 43, Lng: 141, TS: 1}}); got != 0 {
		t.Errorf("1点の PathLength = %f, want 0", got)
	}
}
