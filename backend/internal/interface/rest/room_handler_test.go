package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type testRoomRepository struct {
	rooms             map[string]domain.Room
	members           map[string]map[int64]domain.Member
	ownerUserIDs      map[string]int64
	routes            map[string]map[int64]int64
	activeRoutes      map[string]map[int64]bool
	nextRoomID        int64
	nextMemID         int64
	createCalls       int
	codeConflictsLeft int
}

func TestJoinRoomTrimsDisplayNameBeforePersisting(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	owner := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	joinerCookie := registerTestUser(t, router, "joiner_1")
	response := requestWithCookie(
		router,
		postJSON("/api/rooms/"+created.Room.Code+"/join", `{"displayName":"  walker  ","color":"#ffffff"}`),
		joinerCookie,
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"displayName":"walker"`) {
		t.Fatalf("join response = %d %s, want trimmed display name", response.Code, response.Body)
	}
	if member := repository.members[created.Room.Code][2]; member.DisplayName != "walker" {
		t.Fatalf("persisted display name = %q, want trimmed name", member.DisplayName)
	}
}

func TestJoinRoomValidatesDisplayNameLengthAfterTrimming(t *testing.T) {
	router, _ := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	owner := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	joinerCookie := registerTestUser(t, router, "joiner_1")
	displayName := strings.Repeat(" ", 15) + strings.Repeat("a", 17) + " "
	body, err := json.Marshal(joinRoomRequest{DisplayName: displayName, Color: "#ffffff"})
	if err != nil {
		t.Fatal(err)
	}
	response := requestWithCookie(
		router,
		postJSON("/api/rooms/"+created.Room.Code+"/join", string(body)),
		joinerCookie,
	)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_display_name"`) {
		t.Fatalf("join response = %d %s, want invalid_display_name after trimming", response.Code, response.Body)
	}
}

func TestJoinRoomAllowsClosedRoomForReadOnlyViewing(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	createdResponse := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	closedAt := time.Now()
	room := repository.rooms[created.Room.Code]
	room.ClosedAt = &closedAt
	repository.rooms[created.Room.Code] = room

	joinerCookie := registerTestUser(t, router, "joiner_1")
	response := requestWithCookie(
		router,
		postJSON("/api/rooms/"+created.Room.Code+"/join", `{"displayName":"walker","color":"#ffffff"}`),
		joinerCookie,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("closed room join response = %d %s", response.Code, response.Body)
	}
	if len(repository.members[created.Room.Code]) != 2 {
		t.Fatalf("closed room join did not add the viewer: %+v", repository.members[created.Room.Code])
	}
}

func newTestRoomRepository() *testRoomRepository {
	return &testRoomRepository{
		rooms:        make(map[string]domain.Room),
		members:      make(map[string]map[int64]domain.Member),
		ownerUserIDs: make(map[string]int64),
		routes:       make(map[string]map[int64]int64),
		activeRoutes: make(map[string]map[int64]bool),
	}
}

func (r *testRoomRepository) Create(_ context.Context, owner domain.User, code, title string) (domain.Room, domain.Member, error) {
	r.createCalls++
	if r.codeConflictsLeft > 0 {
		r.codeConflictsLeft--
		return domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
	}
	if _, exists := r.rooms[code]; exists {
		return domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
	}
	r.nextRoomID++
	r.nextMemID++
	room := domain.Room{
		ID:            r.nextRoomID,
		Code:          code,
		Title:         title,
		OwnerUserID:   owner.ID,
		OwnerMemberID: r.nextMemID,
	}
	member := domain.Member{ID: r.nextMemID, DisplayName: owner.Username, Color: domain.FirstMemberColor}
	r.rooms[code] = room
	r.members[code] = map[int64]domain.Member{owner.ID: member}
	r.ownerUserIDs[code] = owner.ID
	return room, member, nil
}

func (r *testRoomRepository) FindByCode(_ context.Context, code string) (domain.Room, error) {
	room, exists := r.rooms[code]
	if !exists {
		return domain.Room{}, domain.ErrRoomNotFound
	}
	return room, nil
}

func (r *testRoomRepository) Snapshot(_ context.Context, code string, userID int64) (domain.RoomSnapshot, domain.Member, error) {
	room, exists := r.rooms[code]
	if !exists {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrRoomNotFound
	}
	if room.IsPrivate && room.OwnerUserID != userID {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrPrivateRoom
	}
	you, exists := r.members[code][userID]
	if !exists {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrMemberNotFound
	}
	if you.BannedAt != nil {
		return domain.RoomSnapshot{}, domain.Member{}, domain.ErrMemberBanned
	}
	snapshot := domain.RoomSnapshot{
		Room:    room,
		Members: make([]domain.Member, 0, len(r.members[code])),
		Routes:  make([]domain.Route, 0, len(r.routes[code])),
	}
	for _, member := range r.members[code] {
		snapshot.Members = append(snapshot.Members, member)
	}
	for id, memberID := range r.routes[code] {
		snapshot.Routes = append(snapshot.Routes, domain.Route{
			ID:       id,
			MemberID: memberID,
			Points:   make([]domain.Point, 0),
		})
	}
	return snapshot, you, nil
}

func (r *testRoomRepository) Join(_ context.Context, code string, userID int64, displayName, color string) (domain.Room, domain.Member, bool, error) {
	room, exists := r.rooms[code]
	if !exists {
		return domain.Room{}, domain.Member{}, false, domain.ErrRoomNotFound
	}
	if room.IsPrivate && room.OwnerUserID != userID {
		return domain.Room{}, domain.Member{}, false, domain.ErrPrivateRoom
	}
	if member, exists := r.members[code][userID]; exists {
		if member.BannedAt != nil {
			return domain.Room{}, domain.Member{}, false, domain.ErrMemberBanned
		}
		return room, member, false, nil
	}
	r.nextMemID++
	member := domain.Member{ID: r.nextMemID, DisplayName: displayName, Color: color}
	r.members[code][userID] = member
	return room, member, true, nil
}

func (r *testRoomRepository) Close(_ context.Context, code string, userID int64) (time.Time, error) {
	room, ok := r.rooms[code]
	if !ok {
		return time.Time{}, domain.ErrRoomNotFound
	}
	if userID != r.ownerUserIDs[code] {
		return time.Time{}, domain.ErrForbidden
	}
	if room.ClosedAt == nil {
		now := time.Now()
		room.ClosedAt = &now
		r.rooms[code] = room
	}
	for routeID := range r.activeRoutes[code] {
		r.activeRoutes[code][routeID] = false
	}
	return *room.ClosedAt, nil
}

func (r *testRoomRepository) Reopen(_ context.Context, code string, userID int64) error {
	room, ok := r.rooms[code]
	if !ok {
		return domain.ErrRoomNotFound
	}
	if userID != r.ownerUserIDs[code] {
		return domain.ErrForbidden
	}
	room.ClosedAt = nil
	r.rooms[code] = room
	return nil
}

func (r *testRoomRepository) Ban(_ context.Context, code string, userID, memberID int64) error {
	return r.setBanned(code, userID, memberID, true)
}

func (r *testRoomRepository) Unban(_ context.Context, code string, userID, memberID int64) error {
	return r.setBanned(code, userID, memberID, false)
}

func (r *testRoomRepository) setBanned(code string, userID, memberID int64, banned bool) error {
	_, ok := r.rooms[code]
	if !ok {
		return domain.ErrRoomNotFound
	}
	if userID != r.ownerUserIDs[code] {
		return domain.ErrForbidden
	}
	for memberUserID, member := range r.members[code] {
		if member.ID != memberID {
			continue
		}
		if memberUserID == r.ownerUserIDs[code] {
			return domain.ErrCannotBanOwner
		}
		if banned {
			now := time.Now()
			member.BannedAt = &now
		} else {
			member.BannedAt = nil
		}
		r.members[code][memberUserID] = member
		return nil
	}
	return domain.ErrMemberNotFound
}

func (r *testRoomRepository) DeleteRoute(_ context.Context, code string, userID, routeID int64) error {
	_, ok := r.rooms[code]
	if !ok {
		return domain.ErrRoomNotFound
	}
	routeUserID, ok := r.routes[code][routeID]
	if !ok {
		return domain.ErrRouteNotFound
	}
	if userID != r.ownerUserIDs[code] && userID != routeUserID {
		return domain.ErrForbidden
	}
	delete(r.routes[code], routeID)
	delete(r.activeRoutes[code], routeID)
	return nil
}

func newRoomTestRouter() (*http.ServeMux, *testRoomRepository) {
	repository := newTestRoomRepository()
	return NewRouter(
		newTestAuthService(),
		usecase.NewRoomService(repository),
		testCookies,
	), repository
}

func registerTestUser(t *testing.T, router *http.ServeMux, username string) *http.Cookie {
	t.Helper()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON(
		"/api/auth/register",
		`{"username":"`+username+`","password":"password123"}`,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("register cookies = %+v", cookies)
	}
	return cookies[0]
}

func requestWithCookie(router http.Handler, request *http.Request, cookie *http.Cookie) *httptest.ResponseRecorder {
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCreateRoomPersistsRoomAndOwnerMember(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")

	response := requestWithCookie(router, postJSON("/api/rooms", `{"title":"公園を歩こう"}`), ownerCookie)
	if response.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body)
	}
	var result struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Room.ID == 0 || result.Room.Title != "公園を歩こう" || result.Room.OwnerMemberID == 0 {
		t.Fatalf("created room = %+v", result.Room)
	}
	if !regexp.MustCompile(`^[A-HJ-NP-Z2-9]{6}$`).MatchString(result.Room.Code) {
		t.Fatalf("room code = %q, want a valid six-character code", result.Room.Code)
	}
	owner, exists := repository.members[result.Room.Code][1]
	if !exists || owner.ID != result.Room.OwnerMemberID || owner.DisplayName != "owner_1" || owner.Color != domain.FirstMemberColor {
		t.Fatalf("owner member = %+v, exists = %t", owner, exists)
	}

	get := httptest.NewRecorder()
	router.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/rooms/"+strings.ToLower(result.Room.Code), nil))
	if get.Code != http.StatusOK ||
		!strings.Contains(get.Body.String(), `"title":"公園を歩こう"`) ||
		!strings.Contains(get.Body.String(), `"ownerMemberId":`) {
		t.Fatalf("get room response = %d %s", get.Code, get.Body)
	}
}

func TestPrivateRoomRejectsOtherMembers(t *testing.T) {
	router, repository := newRoomTestRouter()
	registerTestUser(t, router, "owner_1")
	owner := domain.User{ID: 1, Username: "owner_1"}
	room, _, err := repository.Create(context.Background(), owner, "PRAV23", "owner_1のルーム")
	if err != nil {
		t.Fatal(err)
	}
	room.IsPrivate = true
	repository.rooms[room.Code] = room

	otherCookie := registerTestUser(t, router, "other_1")
	response := requestWithCookie(
		router,
		postJSON("/api/rooms/"+room.Code+"/join", `{"displayName":"other","color":"#abcdef"}`),
		otherCookie,
	)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":"private_room"`) {
		t.Fatalf("private room join = %d %s; want private_room 403", response.Code, response.Body)
	}
	if len(repository.members[room.Code]) != 1 {
		t.Fatalf("private room member count = %d; want owner only", len(repository.members[room.Code]))
	}
}

func TestGetPrivateRoomHidesItsExistence(t *testing.T) {
	router, repository := newRoomTestRouter()
	registerTestUser(t, router, "owner_1")
	room, _, err := repository.Create(context.Background(), domain.User{ID: 1, Username: "owner_1"}, "PRAV23", "owner_1のルーム")
	if err != nil {
		t.Fatal(err)
	}
	room.IsPrivate = true
	repository.rooms[room.Code] = room

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/rooms/"+room.Code, nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"room_not_found"`) {
		t.Fatalf("get private room = %d %s; want room_not_found 404", response.Code, response.Body)
	}
}

func TestRoomCreationRequiresLoginAndValidTitle(t *testing.T) {
	router, _ := newRoomTestRouter()
	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, postJSON("/api/rooms", `{"title":"park"}`))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized create status = %d", unauthorized.Code)
	}
	cookie := registerTestUser(t, router, "owner_1")
	invalidTitle := requestWithCookie(router, postJSON("/api/rooms", `{"title":" "}`), cookie)
	if invalidTitle.Code != http.StatusBadRequest || !strings.Contains(invalidTitle.Body.String(), `"code":"invalid_title"`) {
		t.Fatalf("invalid title response = %d %s", invalidTitle.Code, invalidTitle.Body)
	}
}

func TestCreateRoomTrimsTitleBeforePersisting(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")

	response := requestWithCookie(router, postJSON("/api/rooms", `{"title":"  park  "}`), ownerCookie)
	if response.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body)
	}
	var result struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Room.Title != "park" || repository.rooms[result.Room.Code].Title != "park" {
		t.Fatalf("room title = %q, persisted title = %q; want both trimmed", result.Room.Title, repository.rooms[result.Room.Code].Title)
	}
}

func TestCreateRoomRetriesWhenGeneratedCodeConflicts(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	repository.codeConflictsLeft = 1

	response := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	if response.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", response.Code, response.Body)
	}
	if repository.createCalls != 2 {
		t.Fatalf("repository create calls = %d, want 2 after one code conflict", repository.createCalls)
	}
}

func TestCreateRoomReturnsErrorAfterRepeatedCodeConflicts(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	repository.codeConflictsLeft = 5

	response := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"code":"internal_error"`) {
		t.Fatalf("create response = %d %s, want internal error after retry limit", response.Code, response.Body)
	}
	if repository.createCalls != 5 {
		t.Fatalf("repository create calls = %d, want 5", repository.createCalls)
	}
}

func TestGetUnknownRoomReturnsNotFound(t *testing.T) {
	router, _ := newRoomTestRouter()
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/rooms/ABC234", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), `"code":"room_not_found"`) {
		t.Fatalf("unknown room response = %d %s", response.Code, response.Body)
	}
}

func TestJoinRoomPersistsMemberAndIsIdempotent(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	owner := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	if owner.Code != http.StatusOK {
		t.Fatalf("create status = %d %s", owner.Code, owner.Body)
	}
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	joinerCookie := registerTestUser(t, router, "joiner_1")
	path := "/api/rooms/" + created.Room.Code + "/join"
	first := requestWithCookie(router, postJSON(path, `{"displayName":"歩く人","color":"#Ab12Ef"}`), joinerCookie)
	if first.Code != http.StatusOK {
		t.Fatalf("join status = %d, body = %s", first.Code, first.Body)
	}
	var firstResult struct {
		Member memberResponse `json:"member"`
		Room   roomResponse   `json:"room"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstResult); err != nil {
		t.Fatal(err)
	}
	if firstResult.Member.DisplayName != "歩く人" || firstResult.Member.Color != "#ab12ef" || firstResult.Room.Code != created.Room.Code {
		t.Fatalf("join response = %+v", firstResult)
	}
	second := requestWithCookie(router, postJSON(path, `{"displayName":"別の名前","color":"#ffffff"}`), joinerCookie)
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"displayName":"歩く人"`) {
		t.Fatalf("idempotent join response = %d %s", second.Code, second.Body)
	}
	if repository.nextMemID != 2 {
		t.Fatalf("idempotent join created another member; next member ID = %d", repository.nextMemID)
	}
	if count := len(repository.members[created.Room.Code]); count != 2 {
		t.Fatalf("member count = %d, want owner and one joiner", count)
	}
}

func TestJoinRoomValidatesInputsAndRejectsBannedMember(t *testing.T) {
	router, repository := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	owner := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	joinerCookie := registerTestUser(t, router, "joiner_1")
	path := "/api/rooms/" + created.Room.Code + "/join"
	for _, test := range []struct {
		body string
		code string
	}{
		{`{"displayName":" ","color":"#ffffff"}`, "invalid_display_name"},
		{`{"displayName":"walker","color":"red"}`, "invalid_color"},
		{`{"displayName":"walker","color":"#fff"}`, "invalid_color"},
		{`{"displayName":"walker","color":"#12345"}`, "invalid_color"},
	} {
		response := requestWithCookie(router, postJSON(path, test.body), joinerCookie)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
			t.Errorf("join response = %d %s, want error %s", response.Code, response.Body, test.code)
		}
	}

	joiner := requestWithCookie(router, postJSON(path, `{"displayName":"walker","color":"#ffffff"}`), joinerCookie)
	if joiner.Code != http.StatusOK {
		t.Fatalf("join status = %d %s", joiner.Code, joiner.Body)
	}
	member := repository.members[created.Room.Code][2]
	now := time.Now()
	member.BannedAt = &now
	repository.members[created.Room.Code][2] = member
	banned := requestWithCookie(router, postJSON(path, `{"displayName":"walker","color":"#ffffff"}`), joinerCookie)
	if banned.Code != http.StatusForbidden || !strings.Contains(banned.Body.String(), `"code":"banned"`) {
		t.Fatalf("banned join response = %d %s", banned.Code, banned.Body)
	}
}

func TestJoinRoomAcceptsLowercaseHexColor(t *testing.T) {
	router, _ := newRoomTestRouter()
	ownerCookie := registerTestUser(t, router, "owner_1")
	owner := requestWithCookie(router, postJSON("/api/rooms", `{"title":"park"}`), ownerCookie)
	var created struct {
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	joinerCookie := registerTestUser(t, router, "joiner_1")
	response := requestWithCookie(
		router,
		postJSON("/api/rooms/"+created.Room.Code+"/join", `{"displayName":"walker","color":"#abcdef"}`),
		joinerCookie,
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"color":"#abcdef"`) {
		t.Fatalf("join response = %d %s, want accepted lowercase hex color", response.Code, response.Body)
	}
}

func TestRoomRepositoryErrorsAreNotMasked(t *testing.T) {
	repository := newTestRoomRepository()
	service := usecase.NewRoomService(failingRoomRepository{RoomRepository: repository})
	_, err := service.FindByCode(context.Background(), "ABC234")
	if err == nil || errors.Is(err, domain.ErrRoomNotFound) {
		t.Fatalf("FindByCode error = %v, want repository error", err)
	}
}

type failingRoomRepository struct {
	usecase.RoomRepository
}

func (failingRoomRepository) FindByCode(context.Context, string) (domain.Room, error) {
	return domain.Room{}, errors.New("database unavailable")
}
