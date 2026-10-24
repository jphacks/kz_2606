package ws

import "sync"

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

func (r *Registry) connections(roomCode string) []*client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connections := make([]*client, 0, len(r.byRoom[roomCode]))
	for connection := range r.byRoom[roomCode] {
		connections = append(connections, connection)
	}
	return connections
}
