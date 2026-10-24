package domain

type RoomSnapshot struct {
	Room    Room
	Members []Member
	Routes  []Route
}
