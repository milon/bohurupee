package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
)

func TestHomePage(t *testing.T) {
	t.Parallel()

	srv, err := New(listen.Addr{Host: "127.0.0.1", Port: 4190})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	for _, needle := range []string{
		"DEV ONLY",
		"Bohurupee",
		"127.0.0.1:4190",
		"http://127.0.0.1:4190",
		"/favicon.svg",
		"aria-label=\"Bohurupee\"",
		"Alice Admin",
		"Personas",
		"snippet-authorize",
		"snippet-curl",
		"client_id=dev-client",
		"auto=alice",
		"openid-configuration",
		">dev<",
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("home page missing %q\n%s", needle, html)
		}
	}
}

func TestFavicon(t *testing.T) {
	t.Parallel()

	srv, err := New(listen.Addr{Host: "127.0.0.1", Port: 4190})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "image/svg+xml") {
		t.Fatalf("Content-Type = %q, want image/svg+xml", ct)
	}
	if !strings.Contains(string(body), "<svg") {
		t.Fatalf("favicon is not svg: %s", body)
	}
}

func TestHomePageListsConfiguredPersonas(t *testing.T) {
	t.Parallel()

	srv, err := NewWithOptions(Options{
		Addr: listen.Addr{Host: "127.0.0.1", Port: 4190},
		Personas: []oauth.Persona{
			{ID: "bob", Name: "Bob User", Email: "bob@example.com"},
			{ID: "carol", Name: "Carol Reviewer", Email: "carol@example.com"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	for _, needle := range []string{"Bob User", "Carol Reviewer", "?auto=bob"} {
		if !strings.Contains(html, needle) {
			t.Fatalf("home page missing %q\n%s", needle, html)
		}
	}
	if strings.Contains(html, "Alice Admin") {
		t.Fatal("default persona leaked into configured home page")
	}
}
