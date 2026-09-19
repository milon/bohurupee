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
	"github.com/milon/bohurupee/internal/version"
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

	inDocker := envTruthy("BOHURUPEE_IN_DOCKER")
	defaultBind, dockerAllowsNonLoopback := bindDefaults(inDocker)

	configPath := fs.String("config", "", "YAML config path (default: ./bohurupee.yaml if present)")
	bind := fs.String("bind", defaultBind, "address to bind (loopback only by default)")
	port := fs.Int("port", 4190, "TCP port")
	danger := fs.Bool("dangerously-bind-all-interfaces", false, "allow binding a non-loopback address")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(version.String())
		return nil
	}

	cfg, err := config.LoadPath(*configPath)
	if err != nil {
		return err
	}

	host := cfg.Bind
	listenPort := cfg.Port
	bindFromFlag := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "bind":
			host = *bind
			bindFromFlag = true
		case "port":
			listenPort = *port
		}
	})
	host = resolveHost(host, cfg.BindFromFile, bindFromFlag, inDocker)
	if listenPort == 0 {
		listenPort = *port
	}

	addr := listen.Addr{Host: host, Port: listenPort}
	if err := listen.ValidateLoopback(addr.Host, *danger || dockerAllowsNonLoopback); err != nil {
		return err
	}
	if *danger {
		log.Printf("WARNING: --dangerously-bind-all-interfaces is set; binding %s (DEV ONLY)", addr.String())
	}
	if inDocker {
		log.Printf("WARNING: running in Docker (DEV ONLY); listening on %s. Publish the host port on 127.0.0.1, for example -p 127.0.0.1:%d:%d", addr.String(), addr.Port, addr.Port)
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
		Profiles:    cfg.Profiles,
	})
	if err != nil {
		return err
	}

	log.Printf("Bohurupee (DEV ONLY) listening on %s", addr.DisplayURL())
	return http.ListenAndServe(addr.String(), srv.Handler())
}

// resolveHost picks the listen host.
// An explicit --bind wins. A bind key in the config file wins next.
// Inside the container image, the built-in default becomes 0.0.0.0 so
// Docker can publish the port. Otherwise the default stays loopback.
func resolveHost(host string, bindFromFile, bindFromFlag, inDocker bool) string {
	if bindFromFlag || bindFromFile {
		if host != "" {
			return host
		}
	}
	if inDocker {
		return "0.0.0.0"
	}
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

// bindDefaults is loopback unless the process is the container image.
// BOHURUPEE_IN_DOCKER lets the binary listen on 0.0.0.0 so Docker port
// publishing works. The host publish should still be 127.0.0.1.
func bindDefaults(inDocker bool) (host string, allowNonLoopback bool) {
	if inDocker {
		return "0.0.0.0", true
	}
	return "127.0.0.1", false
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "alice":
		return true
	default:
		return false
	}
}
