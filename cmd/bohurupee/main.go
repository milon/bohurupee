package main

import (
	"errors"
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
	if len(args) > 0 && args[0] == "init" {
		return runInit(args[1:])
	}

	fs := flag.NewFlagSet("bohurupee", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: bohurupee [flags]\n       bohurupee init [--config path] [--force]\n\n")
		fs.PrintDefaults()
	}

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
	resolvedConfig := *configPath
	if resolvedConfig == "" {
		if _, err := os.Stat(config.DefaultPath); err == nil {
			resolvedConfig = config.DefaultPath
		}
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
		Addr:          addr,
		AutoApprove:   envTruthy("BOHURUPEE_AUTO_APPROVE"),
		Personas:      cfg.Personas,
		PKCE:          cfg.PKCE,
		Signer:        signer,
		IDToken:       cfg.IDToken,
		Profiles:      cfg.Profiles,
		ConfigPath:    resolvedConfig,
		RefreshTokens: cfg.RefreshTokens,
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

func runInit(args []string) error {
	fs := flag.NewFlagSet("bohurupee init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	path := fs.String("config", config.DefaultPath, "config file to write")
	force := fs.Bool("force", false, "replace an existing config file")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if err := config.Init(*path, *force); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nEdit personas, then start the server:\n\n", *path)
	if *path == config.DefaultPath {
		fmt.Println("  bohurupee")
		fmt.Println("\nIt loads ./bohurupee.yaml automatically.")
		return nil
	}
	fmt.Printf("  bohurupee --config %s\n", *path)
	return nil
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
