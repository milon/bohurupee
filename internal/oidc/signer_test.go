package oidc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/milon/bohurupee/internal/oauth"
)

func TestSignAndVerifyRoundTrip(t *testing.T) {
	t.Parallel()
	s, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	raw, err := s.IDToken(IDTokenInput{
		Issuer:   "http://127.0.0.1:4190/google",
		Audience: "dev-client",
		Nonce:    "n1",
		Now:      now,
		TTL:      time.Hour,
		Provider: "google",
		Persona:  oauth.Alice,
	})
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := s.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	set, err := jwk.Parse(jwks)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jwt.Parse([]byte(raw), jwt.WithKeySet(set), jwt.WithValidate(true), jwt.WithIssuer("http://127.0.0.1:4190/google"), jwt.WithAudience("dev-client"))
	if err != nil {
		t.Fatal(err)
	}
	sub, ok := tok.Subject()
	if !ok || sub != "google:alice" {
		t.Fatalf("sub = %q ok=%v", sub, ok)
	}
	var nonce string
	if err := tok.Get("nonce", &nonce); err != nil || nonce != "n1" {
		t.Fatalf("nonce = %q err=%v", nonce, err)
	}
}

func TestIDTokenIncludesPersonaClaims(t *testing.T) {
	t.Parallel()
	s, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	p := oauth.Alice
	p.Claims = map[string]any{"role": "admin", "iss": "ignored"}
	raw, err := s.IDToken(IDTokenInput{
		Issuer:   "http://127.0.0.1:4190/google",
		Audience: "dev-client",
		Now:      time.Now(),
		TTL:      time.Hour,
		Provider: "google",
		Persona:  p,
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := jwt.ParseInsecure([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	var role string
	if err := tok.Get("role", &role); err != nil || role != "admin" {
		t.Fatalf("role = %q err=%v", role, err)
	}
	iss, _ := tok.Issuer()
	if iss != "http://127.0.0.1:4190/google" {
		t.Fatalf("iss overwritten: %q", iss)
	}
}

func TestLoadOrCreatePersists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "oidc.key")
	s1, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	s2, err := LoadOrCreate(path)
	if err != nil {
		t.Fatal(err)
	}
	j1, _ := s1.JWKS()
	j2, _ := s2.JWKS()
	if string(j1) != string(j2) {
		t.Fatalf("persisted JWKS mismatch\n%s\n%s", j1, j2)
	}
}
