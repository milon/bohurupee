# Bohurupee (বহুরূপী)

Local fake identity provider for social login development.

Any language or framework can run OAuth / OIDC against localhost instead of
real Google, GitHub, Apple, and friends. Laravel Socialite is a planned optional
adapter, not the product boundary.

**Status:** generic OAuth on localhost. **DEV ONLY** — bind to loopback by default.

## Run from source

Requires [Go 1.24+](https://go.dev/dl/). (Recent macOS dyld refuses binaries
without a Mach-O `LC_UUID`; the Go linker started emitting that in 1.24.)

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

## Generic OAuth (any provider slug)

Authorization code flow. Persona is hardcoded Alice until the consent UI exists.

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/{provider}/authorize` | Requires `client_id`, `redirect_uri`, `response_type=code`, `state`. Auto-approve with `?auto=alice` or `BOHURUPEE_AUTO_APPROVE=1`. |
| `POST` | `/{provider}/token` | Form body and/or HTTP Basic. Open client (any secret). Codes are single-use, 2 minute TTL. |
| `GET` | `/{provider}/userinfo` | `Authorization: Bearer …` → generic JSON (`id`/`sub` are `{provider}:alice`). |

Step-by-step curl: [`examples/curl/README.md`](examples/curl/README.md).

## Dependencies

No third-party Go modules. Codes and bearer tokens are random hex in process
memory; they vanish when the binary exits.

| Dependency | Why |
|------------|-----|
| **Go 1.24+** (toolchain) | One static binary, no Node/PHP required to run the IdP. Module path is `github.com/milon/bohurupee`. 1.24+ so darwin builds include `LC_UUID` (required by dyld on recent macOS). |
| **`flag` (stdlib)** | Two flags and a danger override. Cobra/urfave-cli would add a module and docs surface for a small CLI. |
| **`net/http` (stdlib)** | Method-aware patterns (`GET /{$}`, `GET /{provider}/authorize`) cover home + OAuth without a third-party router. |
| **`html/template` (stdlib)** | Home page interpolates the listen address; templates HTML-escape it. |
| **`net` (stdlib)** | Loopback check via `net.ParseIP` / `IP.IsLoopback` (and DNS lookup for hostnames like `localhost`). |
| **`crypto/rand` + `encoding/hex` (stdlib)** | Authorization codes and access tokens. |
| **`encoding/json` (stdlib)** | Token + userinfo JSON. |
| **`net/url` (stdlib)** | Redirect `code`/`state` query on `redirect_uri`. |
| **`sync` / `time` (stdlib)** | In-memory grant store, 2 minute code TTL, 1 hour token TTL. |
| **GoReleaser** (config only) | Stub `.goreleaser.yaml` so later releases can publish darwin/linux/windows without inventing a build matrix later. Not invoked yet. |

## License

MIT
