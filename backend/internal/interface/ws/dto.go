package ws

import (
	"sort"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

// Point のタプル化など、ワイヤ形式への変換をここに集約する。
type pointTuple [4]any

type roomDTO struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Title         string     `json:"title"`
	OwnerMemberID int64      `json:"ownerMemberId"`
	ClosedAt      *time.Time `json:"closedAt"`
}

type youDTO struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
	IsOwner     bool   `json:"isOwner"`
}

type memberDTO struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"displayName"`
	Color       string `json:"color"`
	Online      bool   `json:"online"`
	Banned      bool   `json:"banned"`
	Recording   bool   `json:"recording"`
}

type routeDTO struct {
	ID       int64        `json:"id"`
	MemberID int64        `json:"member"`
	Color    string       `json:"color"`
	EndedAt  *time.Time   `json:"endedAt"`
	Points   []pointTuple `json:"pts"`
}

type statsDTO struct {
	Meters    float64   `json:"meters"`
	Members   int       `json:"members"`
	StartedAt time.Time `json:"startedAt"`
	ElapsedMs int64     `json:"elapsedMs"`
}

type snapshotDTO struct {
	Type    string      `json:"t"`
	Room    roomDTO     `json:"room"`
	You     youDTO      `json:"you"`
	Stats   statsDTO    `json:"stats"`
	Members []memberDTO `json:"members"`
	Routes  []routeDTO  `json:"routes"`
}

func makeSnapshotDTO(snapshot domain.RoomSnapshot, you domain.Member, registry *Registry, now time.Time) snapshotDTO {
	stats := snapshot.Stats(now)
	memberByID := make(map[int64]domain.Member, len(snapshot.Members))
	recordingByMember := make(map[int64]bool)
	for _, member := range snapshot.Members {
		memberByID[member.ID] = member
	}
	for _, route := range snapshot.Routes {
		if route.EndedAt == nil {
			recordingByMember[route.MemberID] = true
		}
	}

	members := make([]memberDTO, 0, len(snapshot.Members))
	for _, member := range snapshot.Members {
		members = append(members, memberDTO{
			ID:          member.ID,
			DisplayName: member.DisplayName,
			Color:       member.Color,
			Online:      registry.IsOnline(snapshot.Room.Code, member.ID),
			Banned:      member.BannedAt != nil,
			Recording:   recordingByMember[member.ID],
		})
	}
	sort.Slice(members, func(i, j int) bool { return members[i].ID < members[j].ID })

	routes := make([]routeDTO, 0, len(snapshot.Routes))
	for _, route := range snapshot.Routes {
		points := make([]pointTuple, 0, len(route.Points))
		for _, point := range route.Points {
			points = append(points, pointTuple{point.Lat, point.Lng, point.TS, point.Acc})
		}
		routes = append(routes, routeDTO{
			ID:       route.ID,
			MemberID: route.MemberID,
			Color:    memberByID[route.MemberID].Color,
			EndedAt:  route.EndedAt,
			Points:   points,
		})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].ID < routes[j].ID })

	return snapshotDTO{
		Type: "snapshot",
		Room: roomDTO{
			ID:            snapshot.Room.ID,
			Code:          snapshot.Room.Code,
			Title:         snapshot.Room.Title,
			OwnerMemberID: snapshot.Room.OwnerMemberID,
			ClosedAt:      snapshot.Room.ClosedAt,
		},
		You: youDTO{
			ID:          you.ID,
			DisplayName: you.DisplayName,
			Color:       you.Color,
			IsOwner:     you.ID == snapshot.Room.OwnerMemberID,
		},
		Stats: statsDTO{
			Meters:    stats.Meters,
			Members:   stats.Members,
			StartedAt: stats.StartedAt,
			ElapsedMs: stats.Elapsed.Milliseconds(),
		},
		Members: members,
		Routes:  routes,
	}
}
