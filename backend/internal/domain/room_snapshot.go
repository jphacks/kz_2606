package domain

import "time"

type RoomSnapshot struct {
	Room    Room
	Members []Member
	Routes  []Route
}

// RoomStats は部屋全体の集計。snapshot に載せてクライアントの初期表示に使う。
type RoomStats struct {
	// Meters は部屋全体の累積移動距離。BAN されたメンバーの route は、
	// フロントが非表示にする（仕様 7章）ので合計にも入れない。
	Meters float64
	// Members は BAN を除いた参加メンバー数。
	Members int
	// StartedAt は部屋が作られた時刻。クライアントはこれを基準に経過時間を進める。
	StartedAt time.Time
	// Elapsed は StartedAt から、締められていればその時刻まで、
	// 進行中なら now までの経過時間。
	Elapsed time.Duration
}

// Stats は snapshot から部屋全体の集計を求める。
func (s RoomSnapshot) Stats(now time.Time) RoomStats {
	banned := make(map[int64]bool, len(s.Members))
	activeMembers := 0
	for _, member := range s.Members {
		if member.BannedAt != nil {
			banned[member.ID] = true
			continue
		}
		activeMembers++
	}

	meters := 0.0
	for _, route := range s.Routes {
		if banned[route.MemberID] {
			continue
		}
		meters += PathLength(route.Points)
	}

	until := now
	if s.Room.ClosedAt != nil {
		until = *s.Room.ClosedAt
	}
	elapsed := until.Sub(s.Room.CreatedAt)
	if elapsed < 0 {
		elapsed = 0
	}

	return RoomStats{
		Meters:    meters,
		Members:   activeMembers,
		StartedAt: s.Room.CreatedAt,
		Elapsed:   elapsed,
	}
}
