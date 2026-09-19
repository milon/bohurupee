package oauth

import (
	"testing"
	"time"
)

func TestStoreReuseAndExpiry(t *testing.T) {
	t.Parallel()

	clock := NewFrozenClock(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC))
	s := NewStore(clock, time.Minute, time.Hour)

	code, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	_, _, persona, err := s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code, "")
	if err != nil {
		t.Fatal(err)
	}
	if persona != "alice" {
		t.Fatalf("persona = %q", persona)
	}

	_, _, _, err = s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code, "")
	if err == nil {
		t.Fatal("expected reused code to fail")
	}

	code2, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	_, _, _, err = s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code2, "")
	if err == nil {
		t.Fatal("expected expired code to fail")
	}
}

func TestStoreProviderIsolation(t *testing.T) {
	t.Parallel()

	s := NewStore(nil, 0, 0)
	code, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := s.ExchangeCode("acme", "dev", "http://127.0.0.1:9/cb", code, ""); err == nil {
		t.Fatal("expected provider mismatch")
	}
}

func TestValidProvider(t *testing.T) {
	t.Parallel()
	for _, slug := range []string{"google", "acme", "GitHub", "a", "ok_1", "my-app"} {
		if !ValidProvider(slug) {
			t.Fatalf("ValidProvider(%q) = false", slug)
		}
	}
	for _, slug := range []string{"", ".well-known", "a/b", " google", "x y"} {
		if ValidProvider(slug) {
			t.Fatalf("ValidProvider(%q) = true", slug)
		}
	}
}

func TestAliceUserinfoID(t *testing.T) {
	t.Parallel()
	info := Alice.Userinfo("google")
	if info.ID != "google:alice" || info.Sub != "google:alice" {
		t.Fatalf("ids = %q %q", info.ID, info.Sub)
	}
	if info.Email != "alice@example.com" || !info.EmailVerified {
		t.Fatalf("email fields: %+v", info)
	}
}
