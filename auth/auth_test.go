package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func newTestAuthManager(t *testing.T) *AuthManager {
	t.Helper()
	dir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return NewAuthManager(dir, logger)
}

func TestNewAuthManager(t *testing.T) {
	am := newTestAuthManager(t)
	if am == nil {
		t.Fatal("NewAuthManager returned nil")
	}
	if am.cookies == nil {
		t.Error("cookies map is nil")
	}
	if am.client == nil {
		t.Error("HTTP client is nil")
	}
}

func TestSetAndGetCookie(t *testing.T) {
	am := newTestAuthManager(t)

	am.SetCookie("SESSDATA", "test-session")
	if got := am.GetCookie("SESSDATA"); got != "test-session" {
		t.Errorf("GetCookie(SESSDATA) = %q, want test-session", got)
	}

	// Overwrite
	am.SetCookie("SESSDATA", "new-session")
	if got := am.GetCookie("SESSDATA"); got != "new-session" {
		t.Errorf("GetCookie(SESSDATA) after overwrite = %q, want new-session", got)
	}

	// Missing cookie
	if got := am.GetCookie("nonexistent"); got != "" {
		t.Errorf("GetCookie(nonexistent) = %q, want empty", got)
	}
}

func TestClearCookies(t *testing.T) {
	am := newTestAuthManager(t)
	am.SetCookie("SESSDATA", "test")
	am.SetCookie("bili_jct", "test-jct")

	am.ClearCookies()
	if am.IsAuthenticated() {
		t.Error("IsAuthenticated should return false after ClearCookies")
	}
	if len(am.cookies) != 0 {
		t.Errorf("cookies map len = %d, want 0 after clear", len(am.cookies))
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name    string
		cookies map[string]string
		want    bool
	}{
		{
			name:    "no cookies",
			cookies: nil,
			want:    false,
		},
		{
			name:    "only SESSDATA",
			cookies: map[string]string{"SESSDATA": "abc"},
			want:    false,
		},
		{
			name:    "only bili_jct",
			cookies: map[string]string{"bili_jct": "abc"},
			want:    false,
		},
		{
			name:    "both essential cookies",
			cookies: map[string]string{"SESSDATA": "abc", "bili_jct": "def"},
			want:    true,
		},
		{
			name: "all cookies present",
			cookies: map[string]string{
				"SESSDATA": "a", "bili_jct": "b", "DedeUserID": "c", "sid": "d",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := newTestAuthManager(t)
			for k, v := range tt.cookies {
				am.SetCookie(k, v)
			}
			if got := am.IsAuthenticated(); got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetCookiesFromString(t *testing.T) {
	am := newTestAuthManager(t)

	err := am.SetCookiesFromString("SESSDATA=abc; bili_jct=def; DedeUserID=123")
	if err != nil {
		t.Fatalf("SetCookiesFromString: %v", err)
	}

	if got := am.GetCookie("SESSDATA"); got != "abc" {
		t.Errorf("SESSDATA = %q, want abc", got)
	}
	if got := am.GetCookie("bili_jct"); got != "def" {
		t.Errorf("bili_jct = %q, want def", got)
	}
	if got := am.GetCookie("DedeUserID"); got != "123" {
		t.Errorf("DedeUserID = %q, want 123", got)
	}

	// Test with empty and malformed entries
	err = am.SetCookiesFromString("; key-without-value; a=b; ; ;; c=d")
	if err != nil {
		t.Fatalf("SetCookiesFromString with malformed: %v", err)
	}
	if got := am.GetCookie("a"); got != "b" {
		t.Errorf("a = %q, want b", got)
	}
	if got := am.GetCookie("c"); got != "d" {
		t.Errorf("c = %q, want d", got)
	}
}

func TestSaveAndLoadCookies(t *testing.T) {
	am := newTestAuthManager(t)
	am.SetCookie("SESSDATA", "saved-session")
	am.SetCookie("bili_jct", "saved-jct")
	am.SetCookie("DedeUserID", "saved-uid")

	if err := am.SaveCookies(); err != nil {
		t.Fatalf("SaveCookies: %v", err)
	}

	// Verify the file was created.
	cookieFile := filepath.Join(am.configDir, "cookies.json")
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		t.Fatal("cookies.json was not created")
	}

	// Create a new auth manager and load.
	am2 := &AuthManager{
		cookies:   make(map[string]string),
		configDir: am.configDir,
		logger:    logrus.New(),
	}
	if err := am2.LoadCookies(); err != nil {
		t.Fatalf("LoadCookies: %v", err)
	}

	if got := am2.GetCookie("SESSDATA"); got != "saved-session" {
		t.Errorf("loaded SESSDATA = %q, want saved-session", got)
	}
	if got := am2.GetCookie("bili_jct"); got != "saved-jct" {
		t.Errorf("loaded bili_jct = %q, want saved-jct", got)
	}
	if got := am2.GetCookie("DedeUserID"); got != "saved-uid" {
		t.Errorf("loaded DedeUserID = %q, want saved-uid", got)
	}
}

func TestLoadCookies_NoFile(t *testing.T) {
	am := newTestAuthManager(t)
	// No cookie file should exist; loading should succeed silently.
	if err := am.LoadCookies(); err != nil {
		t.Errorf("LoadCookies with no file: %v", err)
	}
}

func TestLoadCookies_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cookieFile := filepath.Join(dir, "cookies.json")
	if err := os.WriteFile(cookieFile, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	am := &AuthManager{
		cookies:   make(map[string]string),
		configDir: dir,
		logger:    logrus.New(),
	}
	if err := am.LoadCookies(); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestGetUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Code int      `json:"code"`
			Data UserInfo `json:"data"`
		}{
			Code: 0,
			Data: UserInfo{
				Mid:   123456,
				Name:  "TestUser",
				Face:  "https://example.com/face.jpg",
				Level: 5,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	am := newTestAuthManager(t)
	// Override the client to talk to the test server by rewriting the base URL.
	// We set the client to use a transport that rewrites to the test server.
	transport := &rewriteTransport{base: server.URL}
	am.client = &http.Client{Transport: transport}

	info, err := am.GetUserInfo()
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.Mid != 123456 {
		t.Errorf("Mid = %d, want 123456", info.Mid)
	}
	if info.Name != "TestUser" {
		t.Errorf("Name = %q, want TestUser", info.Name)
	}
	if info.Level != 5 {
		t.Errorf("Level = %d, want 5", info.Level)
	}
}

func TestGetUserInfo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    -101,
			"message": "not logged in",
		})
	}))
	defer server.Close()

	am := newTestAuthManager(t)
	transport := &rewriteTransport{base: server.URL}
	am.client = &http.Client{Transport: transport}

	_, err := am.GetUserInfo()
	if err == nil {
		t.Error("expected error for API error response, got nil")
	}
}

// rewriteTransport rewrites all HTTP requests to a single base URL for testing.
type rewriteTransport struct {
	base string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := t.base + req.URL.Path
	if req.URL.RawQuery != "" {
		u += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequest(req.Method, u, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}
