package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type adminTestRoom struct {
	router      *http.ServeMux
	repository  *testRoomRepository
	code        string
	owner       *http.Cookie
	member      *http.Cookie
	memberID    int64
	ownerMember int64
	memberUser  int64
	ownerUser   int64
}

func setupAdminTestRoom(t *testing.T) adminTestRoom {
	t.Helper()
	router, repository := newRoomTestRouter()
	registerTestUser(t, router, "unrelated_1")
	ownerCookie := registerTestUser(t, router, "owner_1")
	create := requestWithCookie(router, postJSON("/api/rooms", `{"title":"admin test"}`), ownerCookie)
	if create.Code != http.StatusOK {
		t.Fatalf("create room status = %d, body = %s", create.Code, create.Body)
	}
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	memberCookie := registerTestUser(t, router, "member_1")
	join := requestWithCookie(
		router,
		postJSON("/api/rooms/"+created.Room.Code+"/join", `{"displayName":"member","color":"#00e5ff"}`),
		memberCookie,
	)
	if join.Code != http.StatusOK {
		t.Fatalf("join room status = %d, body = %s", join.Code, join.Body)
	}
	var joined struct {
		Member memberResponse `json:"member"`
	}
	if err := json.Unmarshal(join.Body.Bytes(), &joined); err != nil {
		t.Fatal(err)
	}
	ownerUserID := repository.ownerUserIDs[created.Room.Code]
	memberUserID := findTestMemberUserID(repository, created.Room.Code, joined.Member.ID)
	if ownerUserID == created.Room.OwnerMemberID || memberUserID == joined.Member.ID {
		t.Fatal("test setup must use distinct user and member IDs")
	}
	return adminTestRoom{
		router:      router,
		repository:  repository,
		code:        created.Room.Code,
		owner:       ownerCookie,
		member:      memberCookie,
		memberID:    joined.Member.ID,
		ownerMember: created.Room.OwnerMemberID,
		ownerUser:   ownerUserID,
		memberUser:  memberUserID,
	}
}

func findTestMemberUserID(repository *testRoomRepository, code string, memberID int64) int64 {
	for userID, member := range repository.members[code] {
		if member.ID == memberID {
			return userID
		}
	}
	return 0
}

func TestAdminCloseAndReopenRoom(t *testing.T) {
	testRoom := setupAdminTestRoom(t)
	testRoom.repository.routes[testRoom.code] = map[int64]int64{1: 2}
	testRoom.repository.activeRoutes[testRoom.code] = map[int64]bool{1: true}
	path := "/api/rooms/" + testRoom.code

	unauthorized := httptest.NewRecorder()
	testRoom.router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, path+"/close", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized close status = %d", unauthorized.Code)
	}
	forbidden := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, path+"/close", nil), testRoom.member)
	if forbidden.Code != http.StatusForbidden || !strings.Contains(forbidden.Body.String(), `"code":"forbidden"`) {
		t.Fatalf("non-owner close response = %d %s", forbidden.Code, forbidden.Body)
	}

	closed := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, path+"/close", nil), testRoom.owner)
	if closed.Code != http.StatusOK {
		t.Fatalf("close response = %d %s", closed.Code, closed.Body)
	}
	var closeResult struct {
		ClosedAt time.Time `json:"closedAt"`
	}
	if err := json.Unmarshal(closed.Body.Bytes(), &closeResult); err != nil {
		t.Fatal(err)
	}
	if closeResult.ClosedAt.IsZero() || testRoom.repository.rooms[testRoom.code].ClosedAt == nil {
		t.Fatal("close did not set closedAt")
	}
	if testRoom.repository.activeRoutes[testRoom.code][1] {
		t.Fatal("close did not end the active route")
	}

	reopened := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, path+"/reopen", nil), testRoom.owner)
	if reopened.Code != http.StatusOK || testRoom.repository.rooms[testRoom.code].ClosedAt != nil {
		t.Fatalf("reopen response = %d %s", reopened.Code, reopened.Body)
	}
}

func TestAdminBanAndUnbanMember(t *testing.T) {
	testRoom := setupAdminTestRoom(t)
	basePath := "/api/rooms/" + testRoom.code + "/members/"

	forbidden := requestWithCookie(
		testRoom.router,
		httptest.NewRequest(http.MethodPost, basePath+strconv.FormatInt(testRoom.memberID, 10)+"/ban", nil),
		testRoom.member,
	)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("non-owner ban status = %d, body = %s", forbidden.Code, forbidden.Body)
	}

	banPath := basePath + strconv.FormatInt(testRoom.memberID, 10) + "/ban"
	banned := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, banPath, nil), testRoom.owner)
	if banned.Code != http.StatusNoContent || testRoom.repository.members[testRoom.code][testRoom.memberUser].BannedAt == nil {
		t.Fatalf("ban response = %d %s", banned.Code, banned.Body)
	}
	unbanned := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodDelete, banPath, nil), testRoom.owner)
	if unbanned.Code != http.StatusNoContent || testRoom.repository.members[testRoom.code][testRoom.memberUser].BannedAt != nil {
		t.Fatalf("unban response = %d %s", unbanned.Code, unbanned.Body)
	}

	ownerBanPath := basePath + strconv.FormatInt(testRoom.ownerMember, 10) + "/ban"
	ownerBan := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, ownerBanPath, nil), testRoom.owner)
	if ownerBan.Code != http.StatusBadRequest || !strings.Contains(ownerBan.Body.String(), `"code":"cannot_ban_owner"`) {
		t.Fatalf("owner ban response = %d %s", ownerBan.Code, ownerBan.Body)
	}
	missing := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodPost, basePath+"999/ban", nil), testRoom.owner)
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"code":"member_not_found"`) {
		t.Fatalf("missing member ban response = %d %s", missing.Code, missing.Body)
	}
}

func TestAdminDeleteRouteAllowsOwnerOrRouteOwner(t *testing.T) {
	testRoom := setupAdminTestRoom(t)
	testRoom.repository.routes[testRoom.code] = map[int64]int64{41: testRoom.memberUser, 42: testRoom.ownerUser}
	testRoom.repository.activeRoutes[testRoom.code] = map[int64]bool{41: false, 42: false}
	path := "/api/rooms/" + testRoom.code + "/routes/"
	stranger := registerTestUser(t, testRoom.router, "stranger_1")

	forbidden := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodDelete, path+"41", nil), stranger)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("stranger delete status = %d, body = %s", forbidden.Code, forbidden.Body)
	}
	memberDelete := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodDelete, path+"41", nil), testRoom.member)
	if memberDelete.Code != http.StatusNoContent {
		t.Fatalf("route owner delete response = %d %s", memberDelete.Code, memberDelete.Body)
	}
	ownerDelete := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodDelete, path+"42", nil), testRoom.owner)
	if ownerDelete.Code != http.StatusNoContent {
		t.Fatalf("room owner delete response = %d %s", ownerDelete.Code, ownerDelete.Body)
	}
	missing := requestWithCookie(testRoom.router, httptest.NewRequest(http.MethodDelete, path+"999", nil), testRoom.owner)
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"code":"route_not_found"`) {
		t.Fatalf("missing route response = %d %s", missing.Code, missing.Body)
	}
}
