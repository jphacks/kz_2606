package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
)

func TestRegistryTracksMultipleConnectionsAndRoutesMessages(t *testing.T) {
	registry := NewRegistry()
	firstTab := testClient("ROOM01", 3, registry)
	secondTab := testClient("ROOM01", 3, registry)
	otherMember := testClient("ROOM01", 4, registry)
	registry.Add(firstTab)
	registry.Add(secondTab)
	registry.Add(otherMember)

	if !registry.IsOnline("ROOM01", 3) || !registry.IsOnline("ROOM01", 4) {
		t.Fatal("registered members should be online")
	}
	if registry.Remove(firstTab) {
		t.Fatal("member should remain online while another tab is connected")
	}
	registry.SendTo("ROOM01", 3, []byte("direct"))
	if got := <-secondTab.out; string(got) != "direct" {
		t.Fatalf("direct message = %q", got)
	}
	registry.Broadcast("ROOM01", 3, []byte("broadcast"))
	if got := <-otherMember.out; string(got) != "broadcast" {
		t.Fatalf("broadcast message = %q", got)
	}
	select {
	case got := <-secondTab.out:
		t.Fatalf("broadcast was sent to source member: %q", got)
	default:
	}
	if !registry.Remove(secondTab) || registry.IsOnline("ROOM01", 3) {
		t.Fatal("member should go offline after its last tab disconnects")
	}
}

func TestRegistryBroadcastsRoomEvents(t *testing.T) {
	registry := NewRegistry()
	firstTab := testClient("ROOM01", 3, registry)
	secondTab := testClient("ROOM01", 3, registry)
	otherMember := testClient("ROOM01", 4, registry)
	registry.Add(firstTab)
	registry.Add(secondTab)
	registry.Add(otherMember)

	registry.BroadcastMember("ROOM01", domain.Member{
		ID:          3,
		DisplayName: "walker",
		Color:       "#00e5ff",
	}, false, 0)
	for _, connection := range []*client{firstTab, secondTab} {
		event := readEvent(t, connection)
		if event["t"] != "member" {
			t.Fatalf("member event type = %v", event["t"])
		}
		member := event["member"].(map[string]any)
		if member["id"] != float64(3) || member["online"] != true {
			t.Fatalf("member event payload = %+v", member)
		}
	}
	if event := readEvent(t, otherMember); event["member"] == nil {
		t.Fatalf("other member did not receive member event: %+v", event)
	}

	registry.BroadcastLeave("ROOM01", 3)
	if event := readEvent(t, otherMember); event["t"] != "leave" || event["member"] != float64(3) {
		t.Fatalf("leave event = %+v", event)
	}
	assertNoEvent(t, firstTab)
	assertNoEvent(t, secondTab)

	closedAt := time.Date(2026, time.October, 24, 12, 0, 0, 0, time.UTC)
	registry.BroadcastClosed("ROOM01", closedAt, 1)
	for _, connection := range []*client{firstTab, secondTab, otherMember} {
		assertEventType(t, connection, "closed")
	}
	registry.BroadcastReopened("ROOM01", 1)
	for _, connection := range []*client{firstTab, secondTab, otherMember} {
		assertEventType(t, connection, "reopened")
	}
	registry.BroadcastBanned("ROOM01", 3, 1)
	if event := readEvent(t, otherMember); event["t"] != "banned" || event["member"] != float64(3) {
		t.Fatalf("banned event = %+v", event)
	}
	for _, connection := range []*client{firstTab, secondTab} {
		if event := readEvent(t, connection); event["t"] != "error" || event["code"] != "banned" {
			t.Fatalf("banned member error = %+v", event)
		}
		select {
		case <-connection.closeAfter:
		default:
			t.Fatal("banned member connection was not scheduled to close")
		}
	}
	registry.BroadcastUnbanned("ROOM01", 3, 1)
	assertEventType(t, otherMember, "unbanned")
	registry.BroadcastRouteDeleted("ROOM01", 12)
	if event := readEvent(t, otherMember); event["t"] != "route_deleted" || event["route"] != float64(12) {
		t.Fatalf("route_deleted event = %+v", event)
	}
}

func testClient(roomCode string, memberID int64, registry *Registry) *client {
	return &client{
		out:        make(chan []byte, 10),
		done:       make(chan struct{}),
		closeAfter: make(chan struct{}),
		registry:   registry,
		roomCode:   roomCode,
		memberID:   memberID,
	}
}

func readEvent(t *testing.T, connection *client) map[string]any {
	t.Helper()
	message := <-connection.out
	var event map[string]any
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("decode event %q: %v", message, err)
	}
	return event
}

func assertEventType(t *testing.T, connection *client, want string) {
	t.Helper()
	if event := readEvent(t, connection); event["t"] != want {
		t.Fatalf("event type = %v, want %q", event["t"], want)
	}
}

func assertNoEvent(t *testing.T, connection *client) {
	t.Helper()
	select {
	case message := <-connection.out:
		t.Fatalf("unexpected event: %s", message)
	default:
	}
}
