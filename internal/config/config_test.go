package config

import (
	"os"
	"path/filepath"
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
	gh, ok := cfg.Profiles["github"]
	if !ok || gh.Template != "github" {
		t.Fatalf("github profile = %+v ok=%v", gh, ok)
	}
	if !cfg.BindFromFile || cfg.Bind != "127.0.0.1" {
		t.Fatalf("bind from file = %q set=%v", cfg.Bind, cfg.BindFromFile)
	}
}

func TestOverlayKeepsDefaultPersonas(t *testing.T) {
	t.Parallel()
	cfg := Defaults()
	if err := overlayYAML(&cfg, []byte("pkce: required\n")); err != nil {
		t.Fatal(err)
	}
	if cfg.PKCE != oauth.PKCERequired {
		t.Fatalf("pkce = %q", cfg.PKCE)
	}
	if len(cfg.Personas) != 1 || cfg.Personas[0].ID != "alice" {
		t.Fatalf("personas = %+v", cfg.Personas)
	}
	if cfg.BindFromFile {
		t.Fatal("bind was not in the file")
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
	if err := os.WriteFile(path, []byte("pkce: required\n"), 0o644); err != nil {
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
