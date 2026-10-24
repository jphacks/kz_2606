package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/interface/session"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type testUsers struct{}

func (testUsers) Create(context.Context, string, string) (domain.User, error) {
	return domain.User{}, nil
}

func (testUsers) FindByUsername(context.Context, string) (domain.User, string, error) {
	return domain.User{}, "", domain.ErrUserNotFound
}

func (testUsers) FindByID(_ context.Context, id int64) (domain.User, error) {
	if id != 7 {
		return domain.User{}, domain.ErrUserNotFound
	}
	return domain.User{ID: id, Username: "walker"}, nil
}

type testPasswordHasher struct{}

func (testPasswordHasher) Hash(string) (string, error)  { return "", nil }
func (testPasswordHasher) Compare(string, string) error { return nil }
func (testPasswordHasher) DummyHash() string            { return "" }

type testSessionCodec struct{}

func (testSessionCodec) Encode(int64) (string, error) { return "valid", nil }
func (testSessionCodec) Decode(token string) (int64, error) {
	if token != "valid" {
		return 0, usecase.ErrUnauthorized
	}
	return 7, nil
}

type testRoomSnapshotter struct {
	snapshot domain.RoomSnapshot
	you      domain.Member
	err      error
}

func (s testRoomSnapshotter) Snapshot(context.Context, string, int64) (domain.RoomSnapshot, domain.Member, error) {
	return s.snapshot, s.you, s.err
}

func newTestAuthService() *usecase.AuthService {
	return usecase.NewAuthService(testUsers{}, testPasswordHasher{}, testSessionCodec{})
}

func TestHandlerAuthenticatesRoomMemberAndSendsSnapshot(t *testing.T) {
	snapshotter := testRoomSnapshotter{
		snapshot: domain.RoomSnapshot{
			Room: domain.Room{
				ID:            5,
				Code:          "KZ26AB",
				Title:         "Park",
				OwnerMemberID: 1,
			},
			Members: []domain.Member{
				{ID: 1, DisplayName: "owner", Color: "#00e5ff"},
				{ID: 3, DisplayName: "walker", Color: "#ff2d78"},
				{ID: 4, DisplayName: "banned", Color: "#ffe600", BannedAt: timePointerForTest()},
			},
			Routes: []domain.Route{{
				ID:       12,
				MemberID: 1,
				Points: []domain.Point{{
					Lat: 43.0607,
					Lng: 141.3544,
					TS:  1761278400000,
					Acc: 8.2,
				}},
			}},
		},
		you: domain.Member{ID: 3, DisplayName: "walker", Color: "#ff2d78"},
	}
	registry := NewRegistry()
	handler := NewHandler(newTestAuthService(), snapshotter, registry)
	server := httptest.NewServer(handler)
	defer server.Close()

	conn, response, err := websocket.DefaultDialer.Dial(websocketURL(server.URL, "/ws?room=KZ26AB"), http.Header{
		"Cookie": []string{session.CookieName + "=valid"},
	})
	if err != nil {
		t.Fatalf("websocket dial failed: response=%v err=%v", response, err)
	}
	defer conn.Close()

	var snapshot snapshotDTO
	if err := conn.ReadJSON(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Type != "snapshot" || snapshot.Room.Code != "KZ26AB" {
		t.Fatalf("snapshot header = %+v", snapshot)
	}
	if snapshot.You.ID != 3 || snapshot.You.IsOwner {
		t.Fatalf("snapshot you = %+v", snapshot.You)
	}
	if len(snapshot.Members) != 3 || !snapshot.Members[1].Online || !snapshot.Members[2].Banned {
		t.Fatalf("snapshot members = %+v", snapshot.Members)
	}
	if len(snapshot.Routes) != 1 ||
		len(snapshot.Routes[0].Points) != 1 ||
		snapshot.Routes[0].Points[0] != (pointTuple{43.0607, 141.3544, float64(1761278400000), 8.2}) {
		t.Fatalf("snapshot routes = %+v", snapshot.Routes)
	}

	if err := conn.WriteJSON(map[string]string{"t": "start"}); err != nil {
		t.Fatal(err)
	}
	var unsupported struct {
		Type string `json:"t"`
		Code string `json:"code"`
	}
	if err := conn.ReadJSON(&unsupported); err != nil {
		t.Fatal(err)
	}
	if unsupported.Type != "error" || unsupported.Code != "not_implemented" {
		t.Fatalf("unimplemented operation response = %+v", unsupported)
	}
}

func TestHandlerRejectsUnauthorizedAndNonMemberConnections(t *testing.T) {
	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(NewHandler(newTestAuthService(), testRoomSnapshotter{}, NewRegistry()))
		defer server.Close()
		_, response, err := websocket.DefaultDialer.Dial(
			websocketURL(server.URL, "/ws?room=KZ26AB"),
			nil,
		)
		if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("handshake response=%v err=%v; want 401", response, err)
		}
	})
	for name, accessError := range map[string]error{
		"not a member": domain.ErrMemberNotFound,
		"banned":       domain.ErrMemberBanned,
	} {
		t.Run(name, func(t *testing.T) {
			handler := NewHandler(
				newTestAuthService(),
				testRoomSnapshotter{err: accessError},
				NewRegistry(),
			)
			server := httptest.NewServer(handler)
			defer server.Close()
			headers := http.Header{"Cookie": []string{session.CookieName + "=valid"}}
			_, response, err := websocket.DefaultDialer.Dial(
				websocketURL(server.URL, "/ws?room=KZ26AB"),
				headers,
			)
			if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
				t.Fatalf("handshake response=%v err=%v; want 403", response, err)
			}
		})
	}
}

func TestClientReadPumpEnforcesMessageLimit(t *testing.T) {
	snapshotter := testRoomSnapshotter{
		snapshot: domain.RoomSnapshot{Room: domain.Room{Code: "KZ26AB"}},
		you:      domain.Member{ID: 3},
	}
	server := httptest.NewServer(NewHandler(newTestAuthService(), snapshotter, NewRegistry()))
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial(
		websocketURL(server.URL, "/ws?room=KZ26AB"),
		http.Header{"Cookie": []string{session.CookieName + "=valid"}},
	)
	if err != nil {
		t.Fatalf("websocket dial failed: response=%v err=%v", response, err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, make([]byte, maxMessageSize+1)); err != nil {
		t.Fatal(err)
	}
	_, _, err = conn.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) || closeErr.Code != websocket.CloseMessageTooBig {
		t.Fatalf("oversized message close error = %v; want close code %d", err, websocket.CloseMessageTooBig)
	}
}

func websocketURL(serverURL, path string) string {
	parsed, err := url.Parse(serverURL + path)
	if err != nil {
		panic(err)
	}
	parsed.Scheme = "ws"
	return parsed.String()
}

func timePointerForTest() *time.Time {
	now := time.Now()
	return &now
}
