package main

import "testing"

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
	if got := resolveHost("10.0.0.8", true, false, true); got != "10.0.0.8" {
		t.Fatalf("config bind = %s", got)
	}
	if got := resolveHost("192.168.1.2", false, true, true); got != "192.168.1.2" {
		t.Fatalf("flag bind = %s", got)
	}
}
