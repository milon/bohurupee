package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/config"
)

func TestBindDefaults(t *testing.T) {
	t.Parallel()

	host, allow := bindDefaults(false)
	if host != "127.0.0.1" || allow {
		t.Fatalf("host=%s allow=%v, want 127.0.0.1 false", host, allow)
	}
	host, allow = bindDefaults(true)
	if host != "0.0.0.0" || !allow {
		t.Fatalf("docker host=%s allow=%v, want 0.0.0.0 true", host, allow)
	}
}

func TestResolveHost(t *testing.T) {
	t.Parallel()
	if got := resolveHost("127.0.0.1", false, false, false); got != "127.0.0.1" {
		t.Fatalf("default = %s", got)
	}
	if got := resolveHost("127.0.0.1", false, false, true); got != "0.0.0.0" {
		t.Fatalf("docker default = %s", got)
	}
	if got := resolveHost("127.0.0.1", true, false, true); got != "0.0.0.0" {
		t.Fatalf("docker should upgrade loopback bind from config, got %s", got)
	}
	if got := resolveHost("localhost", true, false, true); got != "0.0.0.0" {
		t.Fatalf("docker should upgrade localhost from config, got %s", got)
	}
	if got := resolveHost("10.0.0.8", true, false, true); got != "10.0.0.8" {
		t.Fatalf("config bind = %s", got)
	}
	if got := resolveHost("192.168.1.2", false, true, true); got != "192.168.1.2" {
		t.Fatalf("flag bind = %s", got)
	}
	if got := resolveHost("127.0.0.1", true, true, true); got != "127.0.0.1" {
		t.Fatalf("explicit --bind must win even in docker, got %s", got)
	}
}

func TestDiscoverConfigPath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if got := discoverConfigPath(false); got != "" {
		t.Fatalf("empty dir = %q", got)
	}

	local := filepath.Join(dir, config.DefaultPath)
	if err := os.WriteFile(local, []byte("port: 4190\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := discoverConfigPath(false); got != config.DefaultPath {
		t.Fatalf("local = %q", got)
	}

	abs := filepath.Join(dir, "mounted.yaml")
	if err := os.WriteFile(abs, []byte("port: 4191\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BOHURUPEE_CONFIG", abs)
	if got := discoverConfigPath(true); got != abs {
		t.Fatalf("env = %q want %q", got, abs)
	}
}

func TestInitCommand(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := run([]string{"init"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadPath(config.DefaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Personas) < 2 || cfg.Personas[1].ID != "bob" {
		t.Fatalf("personas = %+v", cfg.Personas)
	}

	if err := run([]string{"init"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite err = %v", err)
	}
	if err := run([]string{"init", "--force"}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"init", "extra"}); err == nil {
		t.Fatal("expected unexpected argument error")
	}

	other := filepath.Join(dir, "custom.yaml")
	if err := run([]string{"init", "--config", other}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatal(err)
	}
}
