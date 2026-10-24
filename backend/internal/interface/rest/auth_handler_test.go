package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jphacks/kz_2606/backend/internal/domain"
	"github.com/jphacks/kz_2606/backend/internal/usecase"
)

type testUsers struct {
	users             map[string]domain.User
	hashes            map[string]string
	rooms             map[string]domain.Room
	members           map[string][]domain.Member
	nextID            int64
	codeConflictsLeft int
}

func newTestAuthService() *usecase.AuthService {
	return usecase.NewAuthService(&testUsers{
		users:   make(map[string]domain.User),
		hashes:  make(map[string]string),
		rooms:   make(map[string]domain.Room),
		members: make(map[string][]domain.Member),
	}, testHasher{}, testSessions{})
}

func (r *testUsers) CreateWithPersonalRoom(
	_ context.Context,
	username, passwordHash, roomCode, roomTitle string,
) (domain.User, domain.Room, domain.Member, error) {
	if r.codeConflictsLeft > 0 {
		r.codeConflictsLeft--
		return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
	}
	if _, exists := r.users[username]; exists {
		return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrUsernameTaken
	}
	if _, exists := r.rooms[roomCode]; exists {
		return domain.User{}, domain.Room{}, domain.Member{}, domain.ErrRoomCodeTaken
	}
	r.nextID++
	user := domain.User{ID: r.nextID, Username: username, CreatedAt: time.Now()}
	r.users[username] = user
	r.hashes[username] = passwordHash
	room := domain.Room{
		ID:          user.ID,
		Code:        roomCode,
		Title:       roomTitle,
		OwnerUserID: user.ID,
		IsPrivate:   true,
	}
	member := domain.Member{
		ID:          user.ID,
		DisplayName: username,
		Color:       domain.FirstMemberColor,
	}
	room.OwnerMemberID = member.ID
	r.rooms[roomCode] = room
	r.members[roomCode] = []domain.Member{member}
	return user, room, member, nil
}

func TestRegisterRetriesPersonalRoomCodeCollision(t *testing.T) {
	users := &testUsers{
		users:             make(map[string]domain.User),
		hashes:            make(map[string]string),
		rooms:             make(map[string]domain.Room),
		members:           make(map[string][]domain.Member),
		codeConflictsLeft: 1,
	}
	router := NewRouter(usecase.NewAuthService(users, testHasher{}, testSessions{}), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/register", `{"username":"taro_1","password":"password123"}`))
	if response.Code != http.StatusOK || len(users.rooms) != 1 || len(users.members) != 1 {
		t.Fatalf("register response = %d %s; rooms=%d members=%d", response.Code, response.Body, len(users.rooms), len(users.members))
	}
}

func (r *testUsers) FindByUsername(_ context.Context, username string) (domain.User, string, error) {
	user, exists := r.users[username]
	if !exists {
		return domain.User{}, "", domain.ErrUserNotFound
	}
	return user, r.hashes[username], nil
}

func (r *testUsers) FindByID(_ context.Context, id int64) (domain.User, error) {
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

type testHasher struct{}

func (testHasher) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (testHasher) Compare(passwordHash, password string) error {
	if passwordHash != "hash:"+password {
		return usecase.ErrPasswordMismatch
	}
	return nil
}

func (testHasher) DummyHash() string { return "hash:" }

type testSessions struct{}

func (testSessions) Encode(userID int64) (string, error) {
	return fmt.Sprintf("user-%d", userID), nil
}

func (testSessions) Decode(token string) (int64, error) {
	var userID int64
	if _, err := fmt.Sscanf(token, "user-%d", &userID); err != nil || userID <= 0 {
		return 0, errors.New("invalid session")
	}
	return userID, nil
}

var testCookies = CookieOptions{Secure: true, Lifetime: 7 * 24 * time.Hour}

func postJSON(target, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestAuthEndpoints(t *testing.T) {
	users := &testUsers{
		users:   make(map[string]domain.User),
		hashes:  make(map[string]string),
		rooms:   make(map[string]domain.Room),
		members: make(map[string][]domain.Member),
	}
	router := NewRouter(usecase.NewAuthService(users, testHasher{}, testSessions{}), nil, testCookies)

	register := httptest.NewRecorder()
	router.ServeHTTP(register, postJSON("/api/auth/register", `{"username":"Taro_1","password":"password123"}`))
	if register.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", register.Code, register.Body)
	}
	var registered struct {
		User apiUser      `json:"user"`
		Room roomResponse `json:"room"`
	}
	if err := json.Unmarshal(register.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered.User.Username != "taro_1" || registered.User.ID != 1 {
		t.Fatalf("registered user = %+v", registered.User)
	}
	if registered.Room.Title != "taro_1のルーム" || !registered.Room.IsPrivate || registered.Room.OwnerMemberID == 0 {
		t.Fatalf("registered room = %+v; want a private personal room", registered.Room)
	}
	createdMembers := users.members[registered.Room.Code]
	if len(createdMembers) != 1 || createdMembers[0].ID != registered.Room.OwnerMemberID {
		t.Fatalf("personal room members = %+v; want only its owner", createdMembers)
	}
	cookies := register.Result().Cookies()
	if len(cookies) != 1 ||
		cookies[0].Name != sessionCookieName ||
		cookies[0].Path != "/" ||
		!cookies[0].HttpOnly ||
		!cookies[0].Secure ||
		cookies[0].SameSite != http.SameSiteLaxMode ||
		cookies[0].MaxAge != int(testCookies.Lifetime.Seconds()) {
		t.Fatalf("registration session cookie = %+v", cookies)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.AddCookie(cookies[0])
	me := httptest.NewRecorder()
	router.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK || !strings.Contains(me.Body.String(), `"username":"taro_1"`) {
		t.Fatalf("me response = %d %s", me.Code, me.Body)
	}

	logout := httptest.NewRecorder()
	router.ServeHTTP(logout, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.Code)
	}
	expiredCookie := logout.Result().Cookies()
	if len(expiredCookie) != 1 ||
		expiredCookie[0].Name != sessionCookieName ||
		expiredCookie[0].Value != "" ||
		expiredCookie[0].Path != "/" ||
		!expiredCookie[0].HttpOnly ||
		!expiredCookie[0].Secure ||
		expiredCookie[0].SameSite != http.SameSiteLaxMode ||
		expiredCookie[0].MaxAge >= 0 {
		t.Fatalf("logout cookie was not expired: %+v", expiredCookie)
	}

	login := httptest.NewRecorder()
	router.ServeHTTP(login, postJSON("/api/auth/login", `{"username":"taro_1","password":"password123"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("login response = %d %s", login.Code, login.Body)
	}
	if loginCookies := login.Result().Cookies(); len(loginCookies) != 1 || loginCookies[0].Name != sessionCookieName {
		t.Fatalf("login session cookie = %+v", loginCookies)
	}
}

func TestAllProtectedRoutesRequireAuth(t *testing.T) {
	router, _ := newRoomTestRouter()
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/auth/me"},
		{http.MethodPost, "/api/rooms"},
		{http.MethodPost, "/api/rooms/KZ26AB/join"},
		{http.MethodPost, "/api/rooms/KZ26AB/close"},
		{http.MethodPost, "/api/rooms/KZ26AB/reopen"},
		{http.MethodPost, "/api/rooms/KZ26AB/members/1/ban"},
		{http.MethodDelete, "/api/rooms/KZ26AB/members/1/ban"},
		{http.MethodDelete, "/api/rooms/KZ26AB/routes/1"},
	}
	for _, testCase := range cases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(testCase.method, testCase.path, nil))
			if response.Code != http.StatusUnauthorized {
				t.Errorf("%s %s = %d %s; want 401",
					testCase.method, testCase.path, response.Code, response.Body)
			}
		})
	}

	public := httptest.NewRecorder()
	router.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/api/rooms/KZ26AB", nil))
	if public.Code == http.StatusUnauthorized {
		t.Errorf("GET /api/rooms/{code} = 401; route should be public")
	}
}

func TestLoginDoesNotRevealUnknownUsername(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/login", `{"username":"missing","password":"password123"}`))
	if response.Code != http.StatusUnauthorized || !strings.Contains(response.Body.String(), `"code":"invalid_credentials"`) {
		t.Fatalf("login response = %d %s", response.Code, response.Body)
	}
}

func TestRegisterRejectsInvalidCredentials(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/register", `{"username":"ab","password":"short"}`))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_username"`) {
		t.Fatalf("register response = %d %s", response.Code, response.Body)
	}
}

func TestRegisterAcceptsSixteenCharacterUsername(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON(
		"/api/auth/register",
		`{"username":"abcdefghijklmnop","password":"password123"}`,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body)
	}
}

func TestRegisterRejectsSeventeenCharacterUsername(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON(
		"/api/auth/register",
		`{"username":"abcdefghijklmnopq","password":"password123"}`,
	))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_username"`) {
		t.Fatalf("register response = %d %s, want invalid_username", response.Code, response.Body)
	}
}

func TestRegisterReportsEmptyPasswordAsInvalidPassword(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/register", `{"username":"taro_1","password":""}`))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":"invalid_password"`) {
		t.Fatalf("register response = %d %s", response.Code, response.Body)
	}
}

func TestRegisterRejectsNonJSONContentType(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	for _, contentType := range []string{"", "text/plain", "application/x-www-form-urlencoded"} {
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/auth/login",
			strings.NewReader(`{"username":"taro_1","password":"password123"}`),
		)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		} else {
			request.Header.Del("Content-Type")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("Content-Type %q: status = %d %s", contentType, response.Code, response.Body)
		}
		if len(response.Result().Cookies()) != 0 {
			t.Fatalf("Content-Type %q set a session cookie", contentType)
		}
	}
}

func TestLoginAcceptsUnknownFields(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, testCookies)
	register := httptest.NewRecorder()
	router.ServeHTTP(register, postJSON("/api/auth/register", `{"username":"taro_1","password":"password123"}`))
	if register.Code != http.StatusOK {
		t.Fatalf("register status = %d %s", register.Code, register.Body)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/login", `{"username":"taro_1","password":"password123","remember":true}`))
	if response.Code != http.StatusOK {
		t.Fatalf("login response = %d %s", response.Code, response.Body)
	}
}

func TestCookieSecureIsConfigurable(t *testing.T) {
	router := NewRouter(newTestAuthService(), nil, CookieOptions{Secure: false, Lifetime: time.Hour})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, postJSON("/api/auth/register", `{"username":"taro_1","password":"password123"}`))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Secure || cookies[0].MaxAge != 3600 {
		t.Fatalf("session cookie = %+v", cookies)
	}
}
