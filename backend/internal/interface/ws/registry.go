package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

// Registry は部屋ごとの接続を管理し、同じメンバーの複数タブを保持する。
type Registry struct {
	mu     sync.RWMutex
	byRoom map[string]map[*client]struct{}
}

func NewRegistry() *Registry {
	return &Registry{byRoom: make(map[string]map[*client]struct{})}
}

func (r *Registry) Add(connection *client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	connections := r.byRoom[connection.roomCode]
	if connections == nil {
		connections = make(map[*client]struct{})
		r.byRoom[connection.roomCode] = connections
	}
	connections[connection] = struct{}{}
}

// Remove reports whether this was the member's final room connection.
func (r *Registry) Remove(connection *client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	connections := r.byRoom[connection.roomCode]
	delete(connections, connection)
	if len(connections) == 0 {
		delete(r.byRoom, connection.roomCode)
		return true
	}
	for other := range connections {
		if other.memberID == connection.memberID {
			return false
		}
	}
	return true
}

func (r *Registry) IsOnline(roomCode string, memberID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for connection := range r.byRoom[roomCode] {
		if connection.memberID == memberID {
			return true
		}
	}
	return false
}

func (r *Registry) SendTo(roomCode string, memberID int64, message []byte) {
	for _, connection := range r.connections(roomCode) {
		if connection.memberID == memberID {
			connection.enqueue(message)
		}
	}
}

func (r *Registry) Broadcast(roomCode string, excludeMemberID int64, message []byte) {
	for _, connection := range r.connections(roomCode) {
		if connection.memberID != excludeMemberID {
			connection.enqueue(message)
		}
	}
}

func (r *Registry) BroadcastMember(roomCode string, member domain.Member, recording bool, excludeMemberID int64) {
	r.broadcastEvent(roomCode, excludeMemberID, memberEventDTO{
		Type: "member",
		Member: makeMemberDTO(
			member,
			r.IsOnline(roomCode, member.ID),
			recording,
		),
	})
}

func (r *Registry) BroadcastRoute(roomCode string, route domain.Route, member domain.Member, excludeMemberID int64) {
	r.broadcastEvent(roomCode, excludeMemberID, struct {
		Type  string `json:"t"`
		Route struct {
			ID        int64     `json:"id"`
			Member    int64     `json:"member"`
			Color     string    `json:"color"`
			StartedAt time.Time `json:"startedAt"`
		} `json:"route"`
	}{
		Type: "route",
		Route: struct {
			ID        int64     `json:"id"`
			Member    int64     `json:"member"`
			Color     string    `json:"color"`
			StartedAt time.Time `json:"startedAt"`
		}{ID: route.ID, Member: route.MemberID, Color: member.Color, StartedAt: route.StartedAt},
	})
}

func (r *Registry) BroadcastPoints(roomCode string, routeID int64, points []domain.Point, excludeMemberID int64) {
	pts := make([]pointTuple, 0, len(points))
	for _, point := range points {
		pts = append(pts, pointTuple{point.Lat, point.Lng, point.TS, point.Acc})
	}
	r.broadcastEvent(roomCode, excludeMemberID, struct {
		Type   string       `json:"t"`
		Route  int64        `json:"route"`
		Points []pointTuple `json:"pts"`
	}{Type: "points", Route: routeID, Points: pts})
}

func (r *Registry) BroadcastRouteEnd(roomCode string, routeID, memberID int64, endedAt time.Time, excludeMemberID int64) {
	r.broadcastEvent(roomCode, excludeMemberID, struct {
		Type    string    `json:"t"`
		Route   int64     `json:"route"`
		Member  int64     `json:"member"`
		EndedAt time.Time `json:"endedAt"`
	}{Type: "route_end", Route: routeID, Member: memberID, EndedAt: endedAt})
}

func (r *Registry) BroadcastRecordingGate(roomCode string, enabled bool) {
	r.broadcastEvent(roomCode, 0, struct {
		Type    string `json:"t"`
		Enabled bool   `json:"enabled"`
	}{Type: "recording_gate", Enabled: enabled})
}

func (r *Registry) BroadcastLeave(roomCode string, memberID int64) {
	r.broadcastEvent(roomCode, memberID, struct {
		Type   string `json:"t"`
		Member int64  `json:"member"`
	}{Type: "leave", Member: memberID})
}

func (r *Registry) BroadcastClosed(roomCode string, closedAt time.Time, by int64) {
	r.broadcastEvent(roomCode, 0, struct {
		Type     string    `json:"t"`
		ClosedAt time.Time `json:"closedAt"`
		By       int64     `json:"by"`
	}{Type: "closed", ClosedAt: closedAt, By: by})
}

func (r *Registry) BroadcastReopened(roomCode string, by int64) {
	r.broadcastEvent(roomCode, 0, struct {
		Type string `json:"t"`
		By   int64  `json:"by"`
	}{Type: "reopened", By: by})
}

func (r *Registry) BroadcastBanned(roomCode string, memberID, by int64) {
	r.broadcastEvent(roomCode, memberID, struct {
		Type   string `json:"t"`
		Member int64  `json:"member"`
		By     int64  `json:"by"`
	}{Type: "banned", Member: memberID, By: by})

	errorPayload, err := json.Marshal(struct {
		Type    string `json:"t"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Type: "error", Code: "banned", Message: "この部屋への参加は禁止されています"})
	if err != nil {
		slog.Error("encode websocket ban error failed", "err", err)
		return
	}
	for _, connection := range r.connections(roomCode) {
		if connection.memberID == memberID {
			connection.disconnectAfter(errorPayload)
		}
	}
}

func (r *Registry) BroadcastUnbanned(roomCode string, memberID, by int64) {
	r.broadcastEvent(roomCode, 0, struct {
		Type   string `json:"t"`
		Member int64  `json:"member"`
		By     int64  `json:"by"`
	}{Type: "unbanned", Member: memberID, By: by})
}

func (r *Registry) BroadcastRouteDeleted(roomCode string, routeID int64) {
	r.broadcastEvent(roomCode, 0, struct {
		Type  string `json:"t"`
		Route int64  `json:"route"`
	}{Type: "route_deleted", Route: routeID})
}

func (r *Registry) broadcastEvent(roomCode string, excludeMemberID int64, event any) {
	message, err := json.Marshal(event)
	if err != nil {
		slog.Error("encode websocket event failed", "room", roomCode, "err", err)
		return
	}
	r.Broadcast(roomCode, excludeMemberID, message)
}

func (r *Registry) connections(roomCode string) []*client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connections := make([]*client, 0, len(r.byRoom[roomCode]))
	for connection := range r.byRoom[roomCode] {
		connections = append(connections, connection)
	}
	return connections
}
