package domain

import "time"

// Route は1メンバーの1回の記録を表す。EndedAt が nil の間は記録中。
type Route struct {
	ID        int64
	MemberID  int64
	StartedAt time.Time
	EndedAt   *time.Time
	Points    []Point
}

// RouteStats は route 1本の集計。UI の「経路の長さ・経過時間」表示に使う。
// 部屋全体の合計が欲しい場合は、クライアントが route の分を足す。
type RouteStats struct {
	// Meters は累積移動距離。GapMillis 以上空いた区間は加算しない。
	Meters float64
	// Elapsed は StartedAt から、記録を止めてあればその時刻まで、
	// 記録中なら now までの経過時間。
	Elapsed time.Duration
}

// Stats は route 1本の集計を求める。
func (r Route) Stats(now time.Time) RouteStats {
	until := now
	if r.EndedAt != nil {
		until = *r.EndedAt
	}
	elapsed := until.Sub(r.StartedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	return RouteStats{
		Meters:  PathLength(r.Points),
		Elapsed: elapsed,
	}
}

// Recording は記録中（編集可）かどうか。
// 部屋の中で記録中の route は同時に1本までという不変条件がある。
// 仕様: DATA_MODEL.md 7章
func (r Route) Recording() bool {
	return r.EndedAt == nil
}
