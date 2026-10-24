package domain

import "math"

// GapMillis を超えて時刻が離れた点の間は距離に数えない。
// 「超えて」は仕様 3章の `> 30秒` と同じで、ちょうど 30 秒は数える。
// クライアント側の GAP_MS と同じ値・同じ境界にしてあり、片方だけ変えると
// サーバーとクライアントで距離がズレる。仕様: DATA_MODEL.md 3章・11章
const GapMillis int64 = 30_000

const earthRadiusMeters = 6371000.0

// Distance は2点間の大円距離をメートルで返す（haversine）。
func Distance(from, to Point) float64 {
	fromLatRadians := from.Lat * math.Pi / 180
	toLatRadians := to.Lat * math.Pi / 180
	deltaLat := toLatRadians - fromLatRadians
	deltaLng := (to.Lng - from.Lng) * math.Pi / 180

	halfChord := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(fromLatRadians)*math.Cos(toLatRadians)*math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Min(1, math.Sqrt(halfChord)))
}

// PathLength は点列の累積距離をメートルで返す。
// GapMillis を超えて空いた区間は、その間に移動を記録できていないので加算しない。
func PathLength(points []Point) float64 {
	total := 0.0
	for i := 1; i < len(points); i++ {
		if points[i].TS-points[i-1].TS > GapMillis {
			continue
		}
		total += Distance(points[i-1], points[i])
	}
	return total
}
