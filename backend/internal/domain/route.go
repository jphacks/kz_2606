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
