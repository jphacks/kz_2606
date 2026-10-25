package ws

import (
	"sort"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

// Point のタプル化など、ワイヤ形式への変換をここに集約する。
type pointTuple [4]any

type roomDTO struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code"`
	Title            string     `json:"title"`
	OwnerMemberID    int64      `json:"ownerMemberId"`
	IsPrivate        bool       `json:"isPrivate"`
	ClosedAt         *time.Time `json:"closedAt"`
	RecordingEnabled bool       `json:"recordingEnabled"`
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
	ID        int64        `json:"id"`
	MemberID  int64        `json:"member"`
	Color     string       `json:"color"`
	StartedAt time.Time    `json:"startedAt"`
	EndedAt   *time.Time   `json:"endedAt"`
	Meters    float64      `json:"meters"`
	ElapsedMs int64        `json:"elapsedMs"`
	Points    []pointTuple `json:"pts"`
}

type snapshotDTO struct {
	Type    string      `json:"t"`
	Room    roomDTO     `json:"room"`
	You     youDTO      `json:"you"`
	Members []memberDTO `json:"members"`
	Routes  []routeDTO  `json:"routes"`
}

type memberEventDTO struct {
	Type   string    `json:"t"`
	Member memberDTO `json:"member"`
}

func makeSnapshotDTO(snapshot domain.RoomSnapshot, you domain.Member, registry *Registry, now time.Time) snapshotDTO {
	memberByID := make(map[int64]domain.Member, len(snapshot.Members))
	recordingByMember := make(map[int64]bool)
	for _, member := range snapshot.Members {
		memberByID[member.ID] = member
	}
	for _, route := range snapshot.Routes {
		if route.Recording() {
			recordingByMember[route.MemberID] = true
		}
	}
	recordingEnabled := recordingByMember[snapshot.Room.OwnerMemberID]

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
		stats := route.Stats(now)
		routes = append(routes, routeDTO{
			ID:        route.ID,
			MemberID:  route.MemberID,
			Color:     memberByID[route.MemberID].Color,
			StartedAt: route.StartedAt,
			EndedAt:   route.EndedAt,
			Meters:    stats.Meters,
			ElapsedMs: stats.Elapsed.Milliseconds(),
			Points:    points,
		})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].ID < routes[j].ID })

	return snapshotDTO{
		Type: "snapshot",
		Room: roomDTO{
			ID:               snapshot.Room.ID,
			Code:             snapshot.Room.Code,
			Title:            snapshot.Room.Title,
			OwnerMemberID:    snapshot.Room.OwnerMemberID,
			IsPrivate:        snapshot.Room.IsPrivate,
			ClosedAt:         snapshot.Room.ClosedAt,
			RecordingEnabled: recordingEnabled,
		},
		You: youDTO{
			ID:          you.ID,
			DisplayName: you.DisplayName,
			Color:       you.Color,
			IsOwner:     you.ID == snapshot.Room.OwnerMemberID,
		},
		Members: members,
		Routes:  routes,
	}
}

func makeMemberDTO(member domain.Member, online, recording bool) memberDTO {
	return memberDTO{
		ID:          member.ID,
		DisplayName: member.DisplayName,
		Color:       member.Color,
		Online:      online,
		Banned:      member.BannedAt != nil,
		Recording:   recording,
	}
}
