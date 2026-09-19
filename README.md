# Bohurupee (বহুরূপী)

Local fake identity provider for social login development.

Any language or framework can run OAuth / OIDC against localhost instead of
real Google, GitHub, Apple, and friends. Laravel Socialite is a planned optional
adapter, not the product boundary.

**Status:** Milestone 0 (hello binary). **DEV ONLY** — bind to loopback by default.

## Run from source

Requires [Go 1.22+](https://go.dev/dl/).

```bash
go run ./cmd/bohurupee
# → http://127.0.0.1:4190
```

Flags:

| Flag | Default | Purpose |
|------|---------|---------|
| `--bind` | `127.0.0.1` | Listen host |
| `--port` | `4190` | Listen port |
| `--dangerously-bind-all-interfaces` | off | Allow a non-loopback bind |

Non-loopback hosts (`0.0.0.0`, LAN IPs, …) are refused unless that last flag is set.

```bash
go test ./...
```

## Dependencies

Choices are recorded per milestone so later upgrades stay intentional.

### M0 — Repo + hello binary

No third-party Go modules. The binary is a local IdP; extra HTTP/CLI libraries
would not pay for themselves until OAuth routes exist.

| Dependency | Why |
|------------|-----|
| **Go 1.22+** (toolchain) | Matches the project plan: one static binary, no Node/PHP required to run the IdP. Module path is `github.com/milon/bohurupee`. |
| **`flag` (stdlib)** | Two flags and a danger override. Cobra/urfave-cli would add a module and docs surface for a hello CLI. |
| **`net/http` (stdlib)** | Plan default. A router (chi, gorilla) is unnecessary for a single `GET /`. Method-aware patterns (`GET /{$}`) are enough. |
| **`html/template` (stdlib)** | Home page interpolates the listen address; templates HTML-escape it. `embed` is deferred until the consent UI (M2). |
| **`net` (stdlib)** | Loopback check via `net.ParseIP` / `IP.IsLoopback` (and DNS lookup for hostnames like `localhost`). |
| **GoReleaser** (config only) | Stub `.goreleaser.yaml` so M7 can publish darwin/linux/windows without inventing build matrix later. Not invoked in M0. |

## License

MIT
