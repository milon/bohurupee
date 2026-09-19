package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/milon/bohurupee/internal/config"
	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "bohurupee: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("bohurupee", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	configPath := fs.String("config", "", "YAML config path (default: ./bohurupee.yaml if present)")
	bind := fs.String("bind", "127.0.0.1", "address to bind (loopback only by default)")
	port := fs.Int("port", 4190, "TCP port")
	danger := fs.Bool("dangerously-bind-all-interfaces", false, "allow binding a non-loopback address")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.LoadPath(*configPath)
	if err != nil {
		return err
	}

	host := cfg.Bind
	listenPort := cfg.Port
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "bind":
			host = *bind
		case "port":
			listenPort = *port
		}
	})
	if host == "" {
		host = *bind
	}
	if listenPort == 0 {
		listenPort = *port
	}

	addr := listen.Addr{Host: host, Port: listenPort}
	if err := listen.ValidateLoopback(addr.Host, *danger); err != nil {
		return err
	}
	if *danger {
		log.Printf("WARNING: --dangerously-bind-all-interfaces is set; binding %s (DEV ONLY)", addr.String())
	}

	signer, err := oidc.LoadOrCreate(oidc.DefaultKeyPath())
	if err != nil {
		return err
	}

	srv, err := server.NewWithOptions(server.Options{
		Addr:        addr,
		AutoApprove: envTruthy("BOHURUPEE_AUTO_APPROVE"),
		Personas:    cfg.Personas,
		PKCE:        cfg.PKCE,
		Signer:      signer,
		IDToken:     cfg.IDToken,
	})
	if err != nil {
		return err
	}

	log.Printf("Bohurupee (DEV ONLY) listening on %s", addr.DisplayURL())
	return http.ListenAndServe(addr.String(), srv.Handler())
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "alice":
		return true
	default:
		return false
	}
}
