package version

import "testing"

func TestFormat(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":       "dev",
		"dev":    "dev",
		"0.1.0":  "v0.1.0",
		"v0.1.0": "v0.1.0",
		" 1.2 ":  "v1.2",
	}
	for in, want := range cases {
		if got := format(in); got != want {
			t.Errorf("format(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStringMatchesVersion(t *testing.T) {
	t.Parallel()
	if String() != format(Version) {
		t.Fatalf("String() = %q, format(Version) = %q", String(), format(Version))
	}
}
