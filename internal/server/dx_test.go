package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
)

func TestCORSLoopbackOnTokenAndDiscovery(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})

	for _, path := range []string{
		"/google/.well-known/openid-configuration",
		"/google/token",
		"/google/userinfo",
	} {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
			t.Fatalf("%s ACAO = %q", path, got)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/google/.well-known/openid-configuration", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("non-loopback origin should not get CORS")
	}

	req = httptest.NewRequest(http.MethodOptions, "/google/authorize", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("authorize should not get CORS")
	}
}

func TestReloadAddsPersona(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bohurupee.yaml")
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`)

	srv := mustServer(t, Options{
		Addr:       listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas:   []oauth.Persona{oauth.Alice},
		ConfigPath: path,
	})

	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"auto":          {"dave"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("before reload unexpected status %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Query().Get("error") != "invalid_request" {
		t.Fatalf("expected unknown persona error, got %s", loc)
	}

	write(`
personas:
  - id: alice
    email: alice@example.com
    name: Alice
  - id: dave
    email: dave@example.com
    name: Dave
`)
	reload := httptest.NewRequest(http.MethodPost, "/__reload", nil)
	reload.RemoteAddr = "127.0.0.1:54321"
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, reload)
	if rec.Code != http.StatusOK {
		t.Fatalf("reload status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	ids, _ := body["personas"].([]any)
	found := false
	for _, id := range ids {
		if id == "dave" {
			found = true
		}
	}
	if !found {
		t.Fatalf("personas = %#v", body["personas"])
	}

	info := userinfoForCode(t, srv, "google", authorizeCodePersona(t, srv, "google", "dave"), "")
	if info.Email != "dave@example.com" || info.Name != "Dave" {
		t.Fatalf("userinfo = %+v", info)
	}
}

func TestReloadRejectsNonLoopback(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	req := httptest.NewRequest(http.MethodPost, "/__reload", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestReloadKeepsPriorConfigOnBadYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "bohurupee.yaml")
	if err := os.WriteFile(path, []byte(`
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := mustServer(t, Options{
		Addr:       listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas:   []oauth.Persona{oauth.Alice},
		ConfigPath: path,
	})
	if err := os.WriteFile(path, []byte("personas: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/__reload", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	info := userinfoForCode(t, srv, "google", authorizeCodePersona(t, srv, "google", "alice"), "")
	if info.ID != "google:alice" {
		t.Fatalf("persona lost after failed reload: %+v", info)
	}
}

func TestRefreshTokensOptIn(t *testing.T) {
	t.Parallel()
	off := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	code := authorizeCodePersona(t, off, "google", "alice")
	body := exchangeTokenVerifier(t, off, "google", code, "", http.StatusOK)
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal([]byte(body), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken != "" {
		t.Fatal("default must not issue refresh_token")
	}

	on := mustServer(t, Options{
		Addr:          listen.Addr{Host: "127.0.0.1", Port: 4190},
		RefreshTokens: true,
	})
	docReq := httptest.NewRequest(http.MethodGet, "/google/.well-known/openid-configuration", nil)
	docRec := httptest.NewRecorder()
	on.Handler().ServeHTTP(docRec, docReq)
	var doc map[string]any
	if err := json.Unmarshal(docRec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	grants, _ := doc["grant_types_supported"].([]any)
	foundRefresh := false
	for _, g := range grants {
		if g == "refresh_token" {
			foundRefresh = true
		}
	}
	if !foundRefresh {
		t.Fatalf("grants = %#v", grants)
	}

	code = authorizeCodePersona(t, on, "google", "alice")
	body = exchangeTokenVerifier(t, on, "google", code, "", http.StatusOK)
	if err := json.Unmarshal([]byte(body), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.RefreshToken == "" || tok.AccessToken == "" {
		t.Fatalf("token = %#v", tok)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tok.RefreshToken},
		"client_id":     {"dev-client"},
	}
	req := httptest.NewRequest(http.MethodPost, "/google/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	on.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d body = %s", rec.Code, rec.Body.String())
	}
	var refreshed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed.AccessToken == "" || refreshed.AccessToken == tok.AccessToken {
		t.Fatalf("expected new access token, got %#v", refreshed)
	}
	if refreshed.RefreshToken != tok.RefreshToken {
		t.Fatalf("refresh token should be reused, got %q", refreshed.RefreshToken)
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/google/userinfo", nil)
	uiReq.Header.Set("Authorization", "Bearer "+refreshed.AccessToken)
	uiRec := httptest.NewRecorder()
	on.Handler().ServeHTTP(uiRec, uiReq)
	if uiRec.Code != http.StatusOK {
		t.Fatalf("userinfo status = %d", uiRec.Code)
	}
}
