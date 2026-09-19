package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oidc"
)

func TestDiscoveryDocument(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})

	req := httptest.NewRequest(http.MethodGet, "/google/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	iss := "http://127.0.0.1:4190/google"
	if doc["issuer"] != iss {
		t.Fatalf("issuer = %v", doc["issuer"])
	}
	for _, key := range []string{"authorization_endpoint", "token_endpoint", "userinfo_endpoint", "jwks_uri"} {
		v, _ := doc[key].(string)
		if !strings.HasPrefix(v, iss+"/") {
			t.Fatalf("%s = %q", key, v)
		}
	}
	if doc["jwks_uri"] != iss+"/jwks" {
		t.Fatalf("jwks_uri = %v", doc["jwks_uri"])
	}
}

func TestJWKSAlias(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	a := getBody(t, srv, "/google/jwks")
	b := getBody(t, srv, "/google/auth/keys")
	if a != b || !strings.Contains(a, `"keys"`) {
		t.Fatalf("jwks mismatch or missing keys:\n%s\n%s", a, b)
	}
}

func TestIDTokenIssuedForOpenIDAndVerifies(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})
	code := authorizeOpenID(t, srv, "google", "n-42")
	body := exchangeToken(t, srv, "google", code, http.StatusOK)
	var tok struct {
		IDToken     string `json:"id_token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(body), &tok); err != nil {
		t.Fatal(err)
	}
	if tok.IDToken == "" {
		t.Fatalf("missing id_token: %s", body)
	}

	jwks := getBody(t, srv, "/google/jwks")
	set, err := jwk.Parse([]byte(jwks))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse([]byte(tok.IDToken),
		jwt.WithKeySet(set),
		jwt.WithValidate(true),
		jwt.WithIssuer("http://127.0.0.1:4190/google"),
		jwt.WithAudience("dev-client"),
	)
	if err != nil {
		t.Fatal(err)
	}
	sub, ok := parsed.Subject()
	if !ok || sub != "google:alice" {
		t.Fatalf("sub = %q", sub)
	}
	var nonce string
	if err := parsed.Get("nonce", &nonce); err != nil || nonce != "n-42" {
		t.Fatalf("nonce = %q err=%v", nonce, err)
	}
}

func TestNoIDTokenWithoutOpenIDScope(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
	})
	code := authorizeCode(t, srv, "google", false)
	body := exchangeToken(t, srv, "google", code, http.StatusOK)
	if strings.Contains(body, `"id_token"`) {
		t.Fatalf("unexpected id_token: %s", body)
	}
}

func TestAlwaysIDTokenWithoutOpenID(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		IDToken:     oidc.IDTokenAlways,
	})
	code := authorizeCode(t, srv, "google", false)
	body := exchangeToken(t, srv, "google", code, http.StatusOK)
	if !strings.Contains(body, `"id_token"`) {
		t.Fatalf("expected id_token: %s", body)
	}
}

func authorizeOpenID(t *testing.T, srv *Server, provider, nonce string) string {
	t.Helper()
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"scope":         {"openid profile email"},
		"nonce":         {nonce},
		"auto":          {"alice"},
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

func getBody(t *testing.T, srv *Server, path string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}
