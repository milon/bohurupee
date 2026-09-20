package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/oauth"
)

func TestLoadExampleYAML(t *testing.T) {
	t.Parallel()

	cfg, err := LoadPath(filepath.Join("..", "..", "bohurupee.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PKCE != oauth.PKCEOptional {
		t.Fatalf("pkce = %q", cfg.PKCE)
	}
	ids := map[string]bool{}
	for _, p := range cfg.Personas {
		ids[p.ID] = true
	}
	for _, want := range []string{"alice", "bob", "carol"} {
		if !ids[want] {
			t.Fatalf("missing persona %s in %#v", want, ids)
		}
	}
	var bob oauth.Persona
	for _, p := range cfg.Personas {
		if p.ID == "bob" {
			bob = p
		}
	}
	if bob.Email != "bob@example.com" || bob.Name != "Bob User" || bob.Nickname != "bob" {
		t.Fatalf("bob = %+v", bob)
	}
	if bob.EmailVerified {
		t.Fatal("bob should be unverified")
	}
	if bob.Claims["role"] != "user" {
		t.Fatalf("bob claims = %#v", bob.Claims)
	}
	var carol oauth.Persona
	for _, p := range cfg.Personas {
		if p.ID == "carol" {
			carol = p
		}
	}
	if carol.Avatar != "" {
		t.Fatalf("carol avatar = %q", carol.Avatar)
	}
	gh, ok := cfg.Profiles["github"]
	if !ok || gh.Template != "github" {
		t.Fatalf("github profile = %+v ok=%v", gh, ok)
	}
	if !cfg.BindFromFile || cfg.Bind != "127.0.0.1" {
		t.Fatalf("bind from file = %q set=%v", cfg.Bind, cfg.BindFromFile)
	}
}

func TestOverlayKeepsDefaultProfiles(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	raw := []byte(`
pkce: required
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`)
	if err := overlayYAML(&cfg, raw, true); err != nil {
		t.Fatal(err)
	}
	if cfg.PKCE != oauth.PKCERequired {
		t.Fatalf("pkce = %q", cfg.PKCE)
	}
	if len(cfg.Personas) != 1 || cfg.Personas[0].ID != "alice" {
		t.Fatalf("personas = %+v", cfg.Personas)
	}
	fb, ok := cfg.Profiles["facebook"]
	if !ok || fb.Endpoints["userinfo"] != "/me" {
		t.Fatalf("default facebook profile = %+v ok=%v", fb, ok)
	}
	if cfg.BindFromFile {
		t.Fatal("bind was not in the file")
	}
}

func TestLoadRequiresPersonas(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "no-personas.yaml")
	if err := os.WriteFile(path, []byte("pkce: required\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPath(path); err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "personas is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestSparseProviderProfileMerge(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if err := overlayYAML(&cfg, []byte(`
personas:
  - id: alice
    email: alice@example.com
    name: Alice
providerProfiles:
  apple:
    protocol:
      response_mode: query
  facebook:
    response:
      custom: true
`), true); err != nil {
		t.Fatal(err)
	}
	apple := cfg.Profiles["apple"]
	if apple.Template != "apple" {
		t.Fatalf("apple template = %q", apple.Template)
	}
	if apple.Protocol.ResponseMode != "query" {
		t.Fatalf("apple response_mode = %q", apple.Protocol.ResponseMode)
	}
	if !apple.Protocol.IDToken {
		t.Fatal("apple id_token default should remain true")
	}
	fb := cfg.Profiles["facebook"]
	if fb.Endpoints["userinfo"] != "/me" {
		t.Fatalf("facebook endpoints = %#v", fb.Endpoints)
	}
	if fb.Response["custom"] != true {
		t.Fatalf("facebook response = %#v", fb.Response)
	}
	if _, ok := cfg.Profiles["github"]; !ok {
		t.Fatal("github default profile dropped")
	}
}

func TestDefaultsMatchStarter(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if cfg.BindFromFile {
		t.Fatal("defaults must not set BindFromFile")
	}
	if len(cfg.Personas) != 3 {
		t.Fatalf("personas = %d", len(cfg.Personas))
	}
	apple := cfg.Profiles["apple"]
	if apple.Protocol.ResponseMode != "form_post" || !apple.Protocol.IDToken {
		t.Fatalf("apple = %+v", apple)
	}
}

func TestRefreshTokensYAML(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if cfg.RefreshTokens {
		t.Fatal("default refreshTokens should be false")
	}
	if err := overlayYAML(&cfg, []byte(`
refreshTokens: true
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`), true); err != nil {
		t.Fatal(err)
	}
	if !cfg.RefreshTokens {
		t.Fatal("expected refreshTokens true")
	}
}

func TestClientsYAML(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if err := overlayYAML(&cfg, []byte(`
openClient: false
clients:
  - id: strict-app
    redirect_uris:
      - http://127.0.0.1:3000/callback
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`), true); err != nil {
		t.Fatal(err)
	}
	if cfg.OpenClient {
		t.Fatal("openClient should be false")
	}
	c, ok := cfg.Clients["strict-app"]
	if !ok || len(c.RedirectURIs) != 1 || c.RedirectURIs[0] != "http://127.0.0.1:3000/callback" {
		t.Fatalf("client = %+v ok=%v", c, ok)
	}
}

func TestLoadRejectsEmptyPersonas(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(path, []byte("personas: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPath(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadPKCERequired(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "pkce.yaml")
	if err := os.WriteFile(path, []byte(`
pkce: required
personas:
  - id: alice
    email: alice@example.com
    name: Alice
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PKCE != oauth.PKCERequired {
		t.Fatalf("pkce = %q", cfg.PKCE)
	}
}
