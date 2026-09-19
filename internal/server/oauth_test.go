package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
)

func TestAuthorizeRequiresAutoApprove(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})

	req := httptest.NewRequest(http.MethodGet, authorizeURL("google", false), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestOAuthHappyPathGoogleAndAcme(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})

	google := completeFlow(t, srv, "google")
	if google.ID != "google:alice" || google.Sub != "google:alice" {
		t.Fatalf("google ids = %+v", google)
	}

	acme := completeFlow(t, srv, "acme")
	if acme.ID != "acme:alice" || acme.Sub != "acme:alice" {
		t.Fatalf("acme ids = %+v", acme)
	}
	if google.Email != "alice@example.com" || google.Name != "Alice Admin" {
		t.Fatalf("generic fields = %+v", google)
	}
}

func TestAuthorizeAutoQuery(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	code := authorizeCode(t, srv, "google", true)
	if code == "" {
		t.Fatal("empty code")
	}
}

func TestTokenRejectsReusedCode(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})
	code := authorizeCode(t, srv, "google", false)
	_ = exchangeToken(t, srv, "google", code, http.StatusOK)
	body := exchangeToken(t, srv, "google", code, http.StatusBadRequest)
	if !strings.Contains(body, "invalid_grant") {
		t.Fatalf("reuse error body = %s", body)
	}
}

func TestTokenRejectsExpiredCode(t *testing.T) {
	t.Parallel()
	clock := oauth.NewFrozenClock(time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC))
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Clock:       clock,
		CodeTTL:     time.Minute,
	})
	code := authorizeCode(t, srv, "google", false)
	clock.Advance(time.Minute)
	body := exchangeToken(t, srv, "google", code, http.StatusBadRequest)
	if !strings.Contains(body, "invalid_grant") {
		t.Fatalf("expired error body = %s", body)
	}
}

func TestTokenAcceptsBasicAuth(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})
	code := authorizeCode(t, srv, "google", false)

	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {"http://127.0.0.1:9999/callback"},
	}
	req := httptest.NewRequest(http.MethodPost, "/google/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("dev-client", "any-secret")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestUserinfoRejectsWrongProviderToken(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})
	code := authorizeCode(t, srv, "google", false)
	tokenJSON := exchangeToken(t, srv, "google", code, http.StatusOK)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/acme/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func completeFlow(t *testing.T, srv *Server, provider string) oauth.Userinfo {
	t.Helper()
	code := authorizeCode(t, srv, provider, false)
	tokenJSON := exchangeToken(t, srv, provider, code, http.StatusOK)
	var tok struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken == "" || !strings.EqualFold(tok.TokenType, "Bearer") || tok.ExpiresIn <= 0 {
		t.Fatalf("token response = %+v", tok)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+provider+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("userinfo status = %d body = %s", rec.Code, rec.Body.String())
	}
	var info oauth.Userinfo
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	return info
}

func authorizeCode(t *testing.T, srv *Server, provider string, queryAuto bool) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, authorizeURL(provider, queryAuto), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d body = %s", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != "state-xyz" {
		t.Fatalf("state = %q loc = %s", u.Query().Get("state"), loc)
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in %s", loc)
	}
	return code
}

func exchangeToken(t *testing.T, srv *Server, provider, code string, wantStatus int) string {
	t.Helper()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"client_id":     {"dev-client"},
		"client_secret": {"ignored"},
	}
	req := httptest.NewRequest(http.MethodPost, "/"+provider+"/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Code != wantStatus {
		t.Fatalf("token status = %d, want %d body = %s", rec.Code, wantStatus, body)
	}
	return string(body)
}

func authorizeURL(provider string, queryAuto bool) string {
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
	}
	if queryAuto {
		q.Set("auto", "alice")
	}
	return "/" + provider + "/authorize?" + q.Encode()
}

func mustServer(t *testing.T, opts Options) *Server {
	t.Helper()
	srv, err := NewWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	return srv
}
