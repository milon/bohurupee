package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
)

func TestAuthorizeShowsConsent(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})

	req := httptest.NewRequest(http.MethodGet, authorizeURL("google", false), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, needle := range []string{"DEV ONLY", "Alice Admin", "auto=alice", "deny=1", ">Deny<", "data-choice", "role=\"listbox\"", "<title>Google — Bohurupee</title>", "Sign in with"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("consent missing %q\n%s", needle, body)
		}
	}
}

func TestConsentFocusesLastPersona(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	req := httptest.NewRequest(http.MethodGet, authorizeURL("google", false), nil)
	req.AddCookie(&http.Cookie{Name: personaCookie, Value: "bob"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	bobOpen := strings.Index(body, "auto=bob")
	if bobOpen < 0 {
		t.Fatal("missing bob continue URL")
	}
	chunk := body[bobOpen:]
	end := strings.Index(chunk, "</a>")
	if end < 0 {
		t.Fatal("unclosed bob link")
	}
	if !strings.Contains(chunk[:end], `tabindex="0"`) {
		t.Fatal("Bob should start focused after last-persona cookie")
	}
	aliceOpen := strings.Index(body, "auto=alice")
	aliceChunk := body[aliceOpen:bobOpen]
	if strings.Contains(aliceChunk, `tabindex="0"`) {
		t.Fatal("Alice should not be the keyboard start when Bob was last used")
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
	if got := tokenErrorCode(t, body); got != "invalid_grant" {
		t.Fatalf("reuse error = %q body = %s", got, body)
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
	if got := tokenErrorCode(t, body); got != "invalid_grant" {
		t.Fatalf("expired error = %q body = %s", got, body)
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

func TestConsentBobThenUserinfo(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	code := authorizeCodePersona(t, srv, "google", "bob")
	info := userinfoForCode(t, srv, "google", code, "")
	if info.ID != "google:bob" || info.Name != "Bob User" {
		t.Fatalf("userinfo = %+v", info)
	}
}

func TestFormPostHitsRedirectReceiver(t *testing.T) {
	t.Parallel()

	var posted url.Values
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		posted = cloneValues(r.PostForm)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(recv.Close)

	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {recv.URL + "/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"response_mode": {"form_post"},
		"auto":          {"alice"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorize status = %d body = %s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()
	action, values := extractCallbackForm(t, html)
	postReq, err := http.NewRequest(http.MethodPost, action, strings.NewReader(values.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := recv.Client().Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if posted.Get("code") == "" || posted.Get("state") != "state-xyz" {
		t.Fatalf("receiver posted = %v html = %s", posted, html)
	}
}

func TestPKCERequiredAndS256(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr: listen.Addr{Host: "127.0.0.1", Port: 4190},
		PKCE: oauth.PKCERequired,
	})

	missing := httptest.NewRequest(http.MethodGet, authorizeURL("google", true), nil)
	missRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(missRec, missing)
	if missRec.Code != http.StatusFound {
		t.Fatalf("missing challenge status = %d body = %s", missRec.Code, missRec.Body.String())
	}
	if got := authorizeError(t, missRec.Header().Get("Location")); got != "invalid_request" {
		t.Fatalf("missing challenge error = %q", got)
	}

	verifier := "pkce-verifier-value-that-is-long-enough"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	q := url.Values{
		"client_id":             {"dev-client"},
		"redirect_uri":          {"http://127.0.0.1:9999/callback"},
		"response_type":         {"code"},
		"state":                 {"state-xyz"},
		"auto":                  {"alice"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d body = %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	body := exchangeTokenVerifier(t, srv, "google", code, "", http.StatusBadRequest)
	if got := tokenErrorCode(t, body); got != "invalid_grant" {
		t.Fatalf("missing verifier error = %q body = %s", got, body)
	}
	_ = exchangeTokenVerifier(t, srv, "google", code, verifier, http.StatusOK)
}

func TestPKCEForbiddenRejectsChallenge(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr: listen.Addr{Host: "127.0.0.1", Port: 4190},
		PKCE: oauth.PKCEForbidden,
	})
	q := url.Values{
		"client_id":      {"dev-client"},
		"redirect_uri":   {"http://127.0.0.1:9999/callback"},
		"response_type":  {"code"},
		"state":          {"state-xyz"},
		"auto":           {"alice"},
		"code_challenge": {"abc"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := authorizeError(t, rec.Header().Get("Location")); got != "invalid_request" {
		t.Fatalf("forbidden pkce error = %q", got)
	}
}

func TestDenyRedirectsAccessDenied(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"deny":          {"1"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if loc.Query().Get("error") != "access_denied" || loc.Query().Get("state") != "state-xyz" {
		t.Fatalf("location = %s", rec.Header().Get("Location"))
	}
	if loc.Query().Get("code") != "" {
		t.Fatalf("deny must not issue a code: %s", rec.Header().Get("Location"))
	}
}

func TestInvalidRedirectURIDoesNotRedirect(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"not-a-url"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestUnsupportedResponseTypeRedirects(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"token"},
		"state":         {"state-xyz"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := authorizeError(t, rec.Header().Get("Location")); got != "unsupported_response_type" {
		t.Fatalf("error = %q", got)
	}
}

func TestPersonaOverlayUnverifiedBob(t *testing.T) {
	t.Parallel()
	bob := oauth.Persona{
		ID:            "bob",
		Email:         "bob@example.com",
		EmailVerified: false,
		Name:          "Bob User",
		Nickname:      "bob",
		Avatar:        "https://api.dicebear.com/9.x/identicon/svg?seed=bob",
		Claims:        map[string]any{"role": "user"},
	}
	alice := oauth.Alice
	alice.Claims = map[string]any{"role": "admin"}
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: []oauth.Persona{alice, bob},
	})
	bobInfo := userinfoMap(t, srv, "google", authorizeCodePersona(t, srv, "google", "bob"))
	if bobInfo["email_verified"] != false || bobInfo["role"] != "user" || bobInfo["id"] != "google:bob" {
		t.Fatalf("bob = %v", bobInfo)
	}
	aliceInfo := userinfoMap(t, srv, "google", authorizeCodePersona(t, srv, "google", "alice"))
	if aliceInfo["email_verified"] != true || aliceInfo["role"] != "admin" {
		t.Fatalf("alice = %v", aliceInfo)
	}
}

func TestLoginAsCookieSkipsConsent(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	login := httptest.NewRequest(http.MethodPost, "/__login", strings.NewReader("persona=bob&provider=google"))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(loginRec, login)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d body = %s", loginRec.Code, loginRec.Body.String())
	}
	var payload struct {
		Persona string `json:"persona"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Persona != "bob" {
		t.Fatalf("login payload = %+v", payload)
	}
	cookies := loginRec.Result().Cookies()

	req := httptest.NewRequest(http.MethodGet, authorizeURL("google", false), nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d body = %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatalf("expected code, got %s", rec.Header().Get("Location"))
	}
	info := userinfoForCode(t, srv, "google", code, "")
	if info.ID != "google:bob" {
		t.Fatalf("userinfo = %+v", info)
	}
}

func TestTokenUnknownCodeJSON(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	body := exchangeToken(t, srv, "google", "nope", http.StatusBadRequest)
	if got := tokenErrorCode(t, body); got != "invalid_grant" {
		t.Fatalf("error = %q body = %s", got, body)
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

func authorizeCodePersona(t *testing.T, srv *Server, provider, persona string) string {
	t.Helper()
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"auto":          {persona},
	}
	req := httptest.NewRequest(http.MethodGet, "/"+provider+"/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status = %d body = %s", rec.Code, rec.Body.String())
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := loc.Query().Get("code")
	if code == "" {
		t.Fatalf("missing code in %s", loc)
	}
	return code
}

func userinfoForCode(t *testing.T, srv *Server, provider, code, verifier string) oauth.Userinfo {
	t.Helper()
	tokenJSON := exchangeTokenVerifier(t, srv, provider, code, verifier, http.StatusOK)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		t.Fatal(err)
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

func exchangeTokenVerifier(t *testing.T, srv *Server, provider, code, verifier string, wantStatus int) string {
	t.Helper()
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"client_id":     {"dev-client"},
		"client_secret": {"ignored"},
	}
	if verifier != "" {
		form.Set("code_verifier", verifier)
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

func examplePersonas() []oauth.Persona {
	return []oauth.Persona{
		oauth.Alice,
		{ID: "bob", Email: "bob@example.com", EmailVerified: true, Name: "Bob User", Nickname: "bob", Avatar: "https://api.dicebear.com/9.x/identicon/svg?seed=bob"},
		{ID: "carol", Email: "carol@example.com", EmailVerified: true, Name: "Carol Reviewer", Nickname: "carol", Avatar: "https://api.dicebear.com/9.x/identicon/svg?seed=carol"},
	}
}

func extractCallbackForm(t *testing.T, page string) (action string, values url.Values) {
	t.Helper()
	actionRe := regexp.MustCompile(`<form[^>]*action="([^"]+)"`)
	am := actionRe.FindStringSubmatch(page)
	if len(am) != 2 {
		t.Fatalf("no form action in %s", page)
	}
	action = html.UnescapeString(am[1])
	values = url.Values{}
	inputRe := regexp.MustCompile(`<input[^>]*name="([^"]+)"[^>]*value="([^"]*)"`)
	for _, m := range inputRe.FindAllStringSubmatch(page, -1) {
		values.Set(html.UnescapeString(m[1]), html.UnescapeString(m[2]))
	}
	if values.Get("code") == "" {
		t.Fatalf("no code input in %s", page)
	}
	return action, values
}

func cloneValues(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vs := range v {
		out[k] = append([]string{}, vs...)
	}
	return out
}

func tokenErrorCode(t *testing.T, body string) string {
	t.Helper()
	var m struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("token error json: %v body = %s", err, body)
	}
	return m.Error
}

func authorizeError(t *testing.T, loc string) string {
	t.Helper()
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("error")
}
