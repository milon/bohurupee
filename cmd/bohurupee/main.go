package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/milon/bohurupee/internal/listen"
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

	bind := fs.String("bind", "127.0.0.1", "address to bind (loopback only by default)")
	port := fs.Int("port", 4190, "TCP port")
	danger := fs.Bool("dangerously-bind-all-interfaces", false, "allow binding a non-loopback address")

	if err := fs.Parse(args); err != nil {
		return err
	}

	addr := listen.Addr{Host: *bind, Port: *port}
	if err := listen.ValidateLoopback(addr.Host, *danger); err != nil {
		return err
	}
	if *danger {
		log.Printf("WARNING: --dangerously-bind-all-interfaces is set; binding %s (DEV ONLY)", addr.String())
	}

	srv, err := server.NewWithOptions(server.Options{
		Addr:        addr,
		AutoApprove: envTruthy("BOHURUPEE_AUTO_APPROVE"),
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
