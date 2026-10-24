package ws

import "testing"

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

func testClient(roomCode string, memberID int64, registry *Registry) *client {
	return &client{
		out:      make(chan []byte, 1),
		done:     make(chan struct{}),
		registry: registry,
		roomCode: roomCode,
		memberID: memberID,
	}
}
