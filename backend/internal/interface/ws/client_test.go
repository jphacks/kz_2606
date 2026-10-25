package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type testRecordingSession struct {
	startErr error
	member   domain.Member
	route    domain.Route
	points   []domain.Point
}

func (s *testRecordingSession) StartRoute(context.Context, string, int64) (domain.Room, domain.Member, domain.Route, error) {
	return domain.Room{Code: "ROOM01"}, s.member, s.route, s.startErr
}

func (s *testRecordingSession) AppendPoints(_ context.Context, _ string, _ int64, points []domain.Point) (int64, domain.Member, error) {
	for _, point := range points {
		if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 || point.TS < 0 || point.Acc < 0 {
			return 0, domain.Member{}, domain.ErrInvalidPoints
		}
	}
	s.points = append(s.points, points...)
	return s.route.ID, s.member, nil
}

func (s *testRecordingSession) StopRoute(context.Context, string, int64) (domain.Room, domain.Member, int64, time.Time, error) {
	return domain.Room{Code: "ROOM01"}, s.member, s.route.ID, time.UnixMilli(1761278403000).UTC(), nil
}

func (s *testRecordingSession) UpdateMember(_ context.Context, _ string, _ int64, name, color *string) (domain.Member, bool, error) {
	if name != nil {
		s.member.DisplayName = *name
	}
	if color != nil {
		s.member.Color = *color
	}
	return s.member, false, nil
}

func (s *testRecordingSession) Leave(context.Context, string, int64) (domain.Member, int64, time.Time, bool, error) {
	return s.member, s.route.ID, time.UnixMilli(1761278403000).UTC(), true, nil
}

func TestClientRecordingOperationsBroadcastEvents(t *testing.T) {
	registry := NewRegistry()
	session := &testRecordingSession{
		member: domain.Member{ID: 3, DisplayName: "walker", Color: "#00e5ff"},
		route:  domain.Route{ID: 42, MemberID: 3, StartedAt: time.Now()},
	}
	source := testClient("ROOM01", 3, registry)
	source.sessions = session
	observer := testClient("ROOM01", 4, registry)
	registry.Add(source)
	registry.Add(observer)

	source.handleInbound([]byte(`{"t":"start"}`))
	routeEvent := readEvent(t, observer)
	routePayload := routeEvent["route"].(map[string]any)
	if routeEvent["t"] != "route" || routePayload["startedAt"] == nil {
		t.Fatalf("route event missing startedAt: %+v", routeEvent)
	}
	assertEventType(t, source, "route")
	assertEventType(t, source, "member")
	assertEventType(t, observer, "member")

	source.handleInbound([]byte(`{"t":"points","pts":[[43.06,141.35,1761278400000,8.2]]}`))
	points := readEvent(t, observer)
	if points["t"] != "points" || points["route"] != float64(42) {
		t.Fatalf("points event = %+v", points)
	}
	if len(session.points) != 1 || session.points[0].TS != 1761278400000 {
		t.Fatalf("persisted points = %+v", session.points)
	}

	source.handleInbound([]byte(`{"t":"member","displayName":"walk","color":"#aabbcc"}`))
	member := readEvent(t, observer)
	memberPayload := member["member"].(map[string]any)
	if member["t"] != "member" || memberPayload["displayName"] != "walk" || memberPayload["color"] != "#aabbcc" {
		t.Fatalf("member event = %+v", member)
	}
	selfMember := readEvent(t, source)
	if selfMember["member"].(map[string]any)["displayName"] != "walk" {
		t.Fatalf("sender did not receive member event: %+v", selfMember)
	}

	source.handleInbound([]byte(`{"t":"stop"}`))
	routeEnd := readEvent(t, observer)
	if routeEnd["t"] != "route_end" || routeEnd["endedAt"] == nil {
		t.Fatalf("route_end event missing endedAt: %+v", routeEnd)
	}
	ownRouteEnd := readEvent(t, source)
	if ownRouteEnd["t"] != "route_end" || ownRouteEnd["endedAt"] == nil {
		t.Fatalf("sender did not receive route_end with endedAt: %+v", ownRouteEnd)
	}
	assertEventType(t, source, "member")
	assertEventType(t, observer, "member")
}

func TestClientPointsValidatesInputAndReportsSessionErrors(t *testing.T) {
	registry := NewRegistry()
	session := &testRecordingSession{
		member: domain.Member{ID: 3, DisplayName: "walker", Color: "#00e5ff"},
		route:  domain.Route{ID: 42, MemberID: 3},
	}
	source := testClient("ROOM01", 3, registry)
	source.sessions = session
	registry.Add(source)

	source.handleInbound([]byte(`{"t":"points","pts":[[91,141.35,1761278400000,8.2]]}`))
	assertErrorCode(t, source, "invalid_points")
	if len(session.points) != 0 {
		t.Fatalf("invalid points were persisted: %+v", session.points)
	}

	session.startErr = domain.ErrRoomClosed
	source.handleInbound([]byte(`{"t":"start"}`))
	assertErrorCode(t, source, "room_closed")

	session.startErr = domain.ErrRouteInUse
	source.handleInbound([]byte(`{"t":"start"}`))
	assertErrorCode(t, source, "route_in_use")

	source.handleInbound([]byte(`{"t":"points","pts":[[43.06,141.35,1761278400000,8.2]]}`))
	if len(session.points) != 1 {
		t.Fatalf("valid points were not passed to session service: %+v", session.points)
	}

	source.handleInbound([]byte(`{"t":"unknown"}`))
	select {
	case message := <-source.out:
		t.Fatalf("unknown message produced a response: %s", message)
	default:
	}
}

func TestClientLeaveClosesActiveRoute(t *testing.T) {
	registry := NewRegistry()
	session := &testRecordingSession{
		member: domain.Member{ID: 3, DisplayName: "walker", Color: "#00e5ff"},
		route:  domain.Route{ID: 42, MemberID: 3},
	}
	source := testClient("ROOM01", 3, registry)
	source.sessions = session
	observer := testClient("ROOM01", 4, registry)
	registry.Add(source)
	registry.Add(observer)

	source.handleLeave()
	routeEnd := readEvent(t, observer)
	if routeEnd["t"] != "route_end" || routeEnd["endedAt"] == nil {
		t.Fatalf("disconnect route_end missing endedAt: %+v", routeEnd)
	}
	assertEventType(t, observer, "member")
	registry.Remove(source)
	registry.BroadcastLeave("ROOM01", source.memberID)
	assertEventType(t, observer, "leave")
}

type failingAppendSession struct {
	testRecordingSession
	err error
}

func (s *failingAppendSession) AppendPoints(context.Context, string, int64, []domain.Point) (int64, domain.Member, error) {
	return 0, domain.Member{}, s.err
}

func TestClientMapsNoActiveRouteError(t *testing.T) {
	registry := NewRegistry()
	source := testClient("ROOM01", 3, registry)
	source.sessions = &failingAppendSession{err: usecase.ErrNoActiveRoute}
	registry.Add(source)

	source.handleInbound([]byte(`{"t":"points","pts":[[43.06,141.35,1761278400000,8.2]]}`))
	assertErrorCode(t, source, "no_active_route")
}

func assertErrorCode(t *testing.T, connection *client, want string) {
	t.Helper()
	message := <-connection.out
	var event struct {
		Type string `json:"t"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("decode error message: %v", err)
	}
	if event.Type != "error" || event.Code != want {
		t.Fatalf("error event = %+v, want code %q", event, want)
	}
}
