package domain

// Activity は、あるユーザーが行った1回分の記録と、その記録が属する部屋を表す。
// 集計単位は DATA_MODEL.md の定義どおり route とする。
type Activity struct {
	RoomCode    string
	RoomTitle   string
	IsPrivate   bool
	MemberCount int
	Route       Route
}
