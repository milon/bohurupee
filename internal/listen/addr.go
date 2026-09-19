package listen

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Addr is a TCP bind host and port.
type Addr struct {
	Host string
	Port int
}

func (a Addr) String() string {
	return net.JoinHostPort(strings.Trim(a.Host, "[]"), strconv.Itoa(a.Port))
}

func (a Addr) DisplayURL() string {
	host := strings.Trim(a.Host, "[]")
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(a.Port))
}

// ValidateLoopback rejects non-loopback hosts unless allowNonLoopback is set.
func ValidateLoopback(host string, allowNonLoopback bool) error {
	if allowNonLoopback {
		return nil
	}
	loopback, err := isLoopbackHost(host)
	if err != nil {
		return err
	}
	if !loopback {
		return fmt.Errorf("refusing to bind %q: not a loopback address (pass --dangerously-bind-all-interfaces to override)", host)
	}
	return nil
}

func isLoopbackHost(host string) (bool, error) {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	if host == "" {
		return false, nil
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback(), nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return false, fmt.Errorf("lookup %q: %w", host, err)
	}
	if len(ips) == 0 {
		return false, fmt.Errorf("lookup %q: no addresses", host)
	}
	for _, ip := range ips {
		if !ip.IsLoopback() {
			return false, nil
		}
	}
	return true, nil
}
