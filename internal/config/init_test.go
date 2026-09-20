package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitWritesLoadableConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bohurupee.yaml")
	if err := Init(path, false); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Personas) != 3 || cfg.Personas[0].ID != "alice" {
		t.Fatalf("personas = %+v", cfg.Personas)
	}
	if _, ok := cfg.Profiles["github"]; !ok {
		t.Fatal("github profile missing")
	}
	if !cfg.BindFromFile || cfg.Bind != "127.0.0.1" || cfg.Port != 4190 {
		t.Fatalf("bind=%q port=%d fromFile=%v", cfg.Bind, cfg.Port, cfg.BindFromFile)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "bohurupee.yaml")
	if err := os.WriteFile(path, []byte("port: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(path, false); err == nil {
		t.Fatal("expected error")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "port: 1\n" {
		t.Fatalf("file changed: %q", got)
	}
	if err := Init(path, true); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 4190 {
		t.Fatalf("port = %d", cfg.Port)
	}
}

func TestStarterMatchesExample(t *testing.T) {
	t.Parallel()

	root, err := os.ReadFile(filepath.Join("..", "..", "bohurupee.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(root), bytes.TrimSpace(defaultYAML)) {
		t.Fatal("internal/config/example.yaml and bohurupee.example.yaml differ")
	}
}
