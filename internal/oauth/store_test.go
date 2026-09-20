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

	got, err := s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.PersonaID != "alice" {
		t.Fatalf("persona = %q", got.PersonaID)
	}

	_, err = s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code, "")
	if err == nil {
		t.Fatal("expected reused code to fail")
	}

	code2, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)
	_, err = s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code2, "")
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
	if _, err := s.ExchangeCode("acme", "dev", "http://127.0.0.1:9/cb", code, ""); err == nil {
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

func TestStoreRefreshOptIn(t *testing.T) {
	t.Parallel()
	s := NewStore(nil, 0, 0)
	code, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice", Scope: "openid"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Refresh != "" {
		t.Fatal("refresh should be off by default")
	}

	s.SetRefreshEnabled(true)
	code2, err := s.IssueCode(IssueCodeParams{Provider: "google", ClientID: "dev", RedirectURI: "http://127.0.0.1:9/cb", PersonaID: "alice", Scope: "openid"})
	if err != nil {
		t.Fatal(err)
	}
	got, err = s.ExchangeCode("google", "dev", "http://127.0.0.1:9/cb", code2, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Refresh == "" {
		t.Fatal("expected refresh_token")
	}
	refreshed, err := s.RefreshAccess("google", "dev", got.Refresh)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Access == "" || refreshed.Access == got.Access {
		t.Fatalf("expected new access token: %#v", refreshed)
	}
	if refreshed.Refresh != got.Refresh {
		t.Fatalf("refresh = %q want %q", refreshed.Refresh, got.Refresh)
	}
	if _, err := s.LookupToken("google", refreshed.Access); err != nil {
		t.Fatal(err)
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
