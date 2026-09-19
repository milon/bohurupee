package ui

import "testing"

func TestProviderLabel(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"google":  "Google",
		"github":  "GitHub",
		"acme":    "Acme",
		"my-app":  "My App",
		"open_id": "Open Id",
	}
	for slug, want := range cases {
		if got := providerLabel(slug); got != want {
			t.Errorf("providerLabel(%q) = %q, want %q", slug, got, want)
		}
	}
}
