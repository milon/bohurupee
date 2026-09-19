package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/listen"
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
	for _, needle := range []string{"DEV ONLY", "Bohurupee", "127.0.0.1:4190", "http://127.0.0.1:4190", "/favicon.svg", "aria-label=\"Bohurupee\""} {
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
