package listen

import "testing"

func TestValidateLoopback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		host             string
		allowNonLoopback bool
		wantErr          bool
	}{
		{name: "ipv4 loopback", host: "127.0.0.1"},
		{name: "ipv6 loopback", host: "::1"},
		{name: "bracket ipv6", host: "[::1]"},
		{name: "all ipv4", host: "0.0.0.0", wantErr: true},
		{name: "all ipv6", host: "::", wantErr: true},
		{name: "empty host", host: "", wantErr: true},
		{name: "lan", host: "192.168.1.10", wantErr: true},
		{name: "danger all ipv4", host: "0.0.0.0", allowNonLoopback: true},
		{name: "danger lan", host: "192.168.1.10", allowNonLoopback: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateLoopback(tc.host, tc.allowNonLoopback)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for host %q", tc.host)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for host %q: %v", tc.host, err)
			}
		})
	}
}

func TestAddrStringAndDisplayURL(t *testing.T) {
	t.Parallel()

	a := Addr{Host: "127.0.0.1", Port: 4190}
	if got, want := a.String(), "127.0.0.1:4190"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got, want := a.DisplayURL(), "http://127.0.0.1:4190"; got != want {
		t.Fatalf("DisplayURL() = %q, want %q", got, want)
	}

	wildcard := Addr{Host: "0.0.0.0", Port: 4190}
	if got, want := wildcard.DisplayURL(), "http://127.0.0.1:4190"; got != want {
		t.Fatalf("wildcard DisplayURL() = %q, want %q", got, want)
	}
}
