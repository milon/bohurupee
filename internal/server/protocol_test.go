package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/config"
	"github.com/milon/bohurupee/internal/listen"
)

func TestPromptLoginIgnoresLastPersonaCookie(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"prompt":        {"login"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: personaCookie, Value: "bob"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (want consent)", rec.Code)
	}
	body := rec.Body.String()
	aliceOpen := strings.Index(body, "auto=alice")
	bobOpen := strings.Index(body, "auto=bob")
	if aliceOpen < 0 || bobOpen < 0 {
		t.Fatal("missing persona links")
	}
	aliceChunk := body[aliceOpen:bobOpen]
	if !strings.Contains(aliceChunk, `tabindex="0"`) {
		t.Fatal("prompt=login should ignore last-persona cookie and focus the first persona")
	}
	bobChunk := body[bobOpen:]
	end := strings.Index(bobChunk, "</a>")
	if end > 0 && strings.Contains(bobChunk[:end], `tabindex="0"`) {
		t.Fatal("Bob should not stay focused under prompt=login")
	}
}

func TestPromptLoginIgnoresAutoCookie(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"prompt":        {"login"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: autoCookie, Value: "bob"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `class="persona`) {
		t.Fatalf("expected consent HTML, got %s", rec.Body.String()[:min(200, rec.Body.Len())])
	}
}

func TestLoginHintHighlightsBob(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:     listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: examplePersonas(),
	})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"login_hint":    {"bob"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	req.AddCookie(&http.Cookie{Name: personaCookie, Value: "carol"})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	bobOpen := strings.Index(body, "auto=bob")
	if bobOpen < 0 {
		t.Fatal("missing bob")
	}
	chunk := body[bobOpen:]
	end := strings.Index(chunk, "</a>")
	if end < 0 || !strings.Contains(chunk[:end], `tabindex="0"`) {
		t.Fatal("login_hint=bob should focus Bob")
	}
}

func TestClientRedirectURIsAllowlist(t *testing.T) {
	t.Parallel()
	falseVal := false
	srv := mustServer(t, Options{
		Addr:       listen.Addr{Host: "127.0.0.1", Port: 4190},
		OpenClient: &falseVal,
		Clients: map[string]config.Client{
			"strict-app": {
				ID:           "strict-app",
				RedirectURIs: []string{"http://127.0.0.1:3000/callback"},
			},
		},
	})

	okQ := url.Values{
		"client_id":     {"strict-app"},
		"redirect_uri":  {"http://127.0.0.1:3000/callback"},
		"response_type": {"code"},
		"state":         {"xyz"},
		"auto":          {"alice"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+okQ.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("allowed redirect status = %d", rec.Code)
	}

	badQ := cloneValues(okQ)
	badQ.Set("redirect_uri", "http://127.0.0.1:9999/surprise")
	req = httptest.NewRequest(http.MethodGet, "/google/authorize?"+badQ.Encode(), nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("surprise redirect status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "redirect_uri") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestOpenClientStillAcceptsAnyRedirect(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr: listen.Addr{Host: "127.0.0.1", Port: 4190},
		Clients: map[string]config.Client{
			"strict-app": {
				ID:           "strict-app",
				RedirectURIs: []string{"http://127.0.0.1:3000/callback"},
			},
		},
	})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"xyz"},
		"auto":          {"alice"},
	}
	req := httptest.NewRequest(http.MethodGet, "/google/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("open client status = %d", rec.Code)
	}
}

func TestDiscoveryClaimsSupported(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{Addr: listen.Addr{Host: "127.0.0.1", Port: 4190}})
	req := httptest.NewRequest(http.MethodGet, "/google/.well-known/openid-configuration", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	claims, ok := doc["claims_supported"].([]any)
	if !ok || len(claims) == 0 {
		t.Fatalf("claims_supported = %#v", doc["claims_supported"])
	}
	found := map[string]bool{}
	for _, c := range claims {
		s, _ := c.(string)
		found[s] = true
	}
	for _, want := range []string{"sub", "given_name", "family_name", "at_hash", "email"} {
		if !found[want] {
			t.Fatalf("missing claim %s in %#v", want, claims)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
