package domain

// Point は緯度・経度・取得時刻（unix milliseconds）・測位精度を表す。
// JSONのワイヤ形式は interface/ws/dto.go で変換する。
type Point struct {
	Lat float64
	Lng float64
	TS  int64
	Acc float32
}
