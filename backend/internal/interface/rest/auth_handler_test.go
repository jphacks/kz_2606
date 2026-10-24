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
	users  map[string]domain.User
	hashes map[string]string
	nextID int64
}

func newTestAuthService() *usecase.AuthService {
	return usecase.NewAuthService(&testUsers{
		users:  make(map[string]domain.User),
		hashes: make(map[string]string),
	}, testHasher{}, testSessions{})
}

func (r *testUsers) Create(_ context.Context, username, passwordHash string) (domain.User, error) {
	if _, exists := r.users[username]; exists {
		return domain.User{}, domain.ErrUsernameTaken
	}
	r.nextID++
	user := domain.User{ID: r.nextID, Username: username, CreatedAt: time.Now()}
	r.users[username] = user
	r.hashes[username] = passwordHash
	return user, nil
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
	router := NewRouter(newTestAuthService(), nil, testCookies)

	register := httptest.NewRecorder()
	router.ServeHTTP(register, postJSON("/api/auth/register", `{"username":"Taro_1","password":"password123"}`))
	if register.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", register.Code, register.Body)
	}
	var registered struct {
		User apiUser `json:"user"`
	}
	if err := json.Unmarshal(register.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered.User.Username != "taro_1" || registered.User.ID != 1 {
		t.Fatalf("registered user = %+v", registered.User)
	}
	cookies := register.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly || !cookies[0].Secure {
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
	if len(expiredCookie) != 1 || expiredCookie[0].MaxAge >= 0 {
		t.Fatalf("logout cookie was not expired: %+v", expiredCookie)
	}

	login := httptest.NewRecorder()
	router.ServeHTTP(login, postJSON("/api/auth/login", `{"username":"taro_1","password":"password123"}`))
	if login.Code != http.StatusOK {
		t.Fatalf("login response = %d %s", login.Code, login.Body)
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
