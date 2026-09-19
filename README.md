<img src="assets/logo.svg" alt="" width="88" align="right">

# Bohurupee (বহুরূপী)

Local fake identity provider for social login development.

Any language or framework can run OAuth / OIDC against localhost instead of
real Google, GitHub, Apple, and friends. Laravel Socialite is an optional
adapter (`milon/bohurupee-laravel`), not the product boundary.

**Status:** v0.1. **DEV ONLY** — loopback by default.

## Install

You do not need Go, Node, or PHP to run Bohurupee. Download a binary, or build from source.

### Release binary

Archives for v0.1.0 are on [GitHub Releases](https://github.com/milon/bohurupee/releases/tag/v0.1.0). Each one contains the `bohurupee` binary and `bohurupee.example.yaml`. `checksums.txt` sits next to the archives — check it before you run the file.

| OS      | Architecture  | File                                  |
|---------|---------------|---------------------------------------|
| macOS   | Apple silicon | `bohurupee_0.1.0_darwin_arm64.tar.gz` |
| macOS   | Intel         | `bohurupee_0.1.0_darwin_amd64.tar.gz` |
| Linux   | x86_64        | `bohurupee_0.1.0_linux_amd64.tar.gz`  |
| Linux   | arm64         | `bohurupee_0.1.0_linux_arm64.tar.gz`  |
| Windows | x86_64        | `bohurupee_0.1.0_windows_amd64.zip`   |

```bash
# example: macOS Apple silicon
curl -fsSL -o bohurupee.tar.gz \
  https://github.com/milon/bohurupee/releases/download/v0.1.0/bohurupee_0.1.0_darwin_arm64.tar.gz
tar -xzf bohurupee.tar.gz
./bohurupee --version
./bohurupee --config ./bohurupee.example.yaml
```

### Docker

`ghcr.io/milon/bohurupee` is a scratch image: the static binary and nothing else (no shell). The image sets `BOHURUPEE_IN_DOCKER=1`, so the process listens on `0.0.0.0` **inside** the container. Publish the host port on loopback only:

```bash
docker run --rm -p 127.0.0.1:4190:4190 ghcr.io/milon/bohurupee:v0.1.0
```

Do not publish `0.0.0.0:4190` on a shared network. A config file with `bind: 127.0.0.1` listens on container loopback, which Docker cannot publish — use `bind: 0.0.0.0` in that file, or omit `--config`.

### From source

Requires [Go 1.25+](https://go.dev/dl/). (Recent macOS dyld refuses binaries
without a Mach-O `LC_UUID`; the Go linker started emitting that in 1.24.)

```bash
go build -o ./bohurupee ./cmd/bohurupee
./bohurupee --version
./bohurupee --config ./bohurupee.example.yaml
```

## Security

**DEV ONLY.** This is a fake identity provider for development on your machine.

- The binary refuses non-loopback binds unless you pass `--dangerously-bind-all-interfaces`.
- The Docker image is the exception: it may bind `0.0.0.0` inside the container. Keep the published host port on `127.0.0.1`.
- Any `client_id` and `client_secret` are accepted. Codes and tokens live in memory and disappear when the process exits.
- Do not point a production app at it, and do not expose it on a LAN or the public internet.

## Quick start

In another terminal, run both curl examples:

```bash
./examples/curl/run-all.sh
```

The first flow returns generic `/acme` userinfo. The second returns
GitHub-shaped userinfo from the same local server. Neither calls a real
provider.

Copy `bohurupee.example.yaml` to `bohurupee.yaml` (or pass `--config`) to change
personas without rebuilding. CLI `--bind` / `--port` override the file.

For browser login, open an authorize URL and pick a persona. Command-line
examples auto-approve with `?auto=alice`.

| Flag                                | Default                       | Purpose                        |
|-------------------------------------|-------------------------------|--------------------------------|
| `--bind`                            | `127.0.0.1`                   | Listen host                    |
| `--port`                            | `4190`                        | Listen port                    |
| `--config`                          | `./bohurupee.yaml` if present | Personas, PKCE mode, bind/port |
| `--version`                         |                               | Print the version and exit     |
| `--dangerously-bind-all-interfaces` | off                           | Allow a non-loopback bind      |

Non-loopback hosts (`0.0.0.0`, LAN IPs, …) are refused unless that last flag is set.

```bash
go test ./...
```

## Generic OAuth (any provider slug)

Authorization code flow. The authorize page lists configured personas. `id` /
`sub` are `{provider}:{persona}`.

| Method | Path                                           | Notes                                                                                                                                                                                                 |
|--------|------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GET`  | `/{provider}/authorize`                        | Requires `client_id`, `redirect_uri`, `response_type=code`, `state`. Consent UI, or `?auto=<persona>` / `BOHURUPEE_AUTO_APPROVE=1`. `response_mode=form_post` posts `code`/`state` to `redirect_uri`. |
| `POST` | `/{provider}/token`                            | Form body and/or HTTP Basic. Open client (any secret). Codes are single-use, 2 minute TTL. PKCE `code_verifier` when a challenge was used.                                                            |
| `GET`  | `/{provider}/userinfo`                         | `Authorization: Bearer …` → generic JSON, or a provider profile if configured                                                                                                                         |
| `GET`  | `/{provider}/.well-known/openid-configuration` | OIDC discovery (issuer, authorize, token, userinfo, jwks)                                                                                                                                             |
| `GET`  | `/{provider}/jwks`                             | JWKS for `id_token` (alias `/{provider}/auth/keys`)                                                                                                                                                   |

PKCE mode in YAML: `optional` (default), `required`, or `forbidden`.
`idToken: openid` (default) issues an RS256 `id_token` when `scope` includes
`openid`; `idToken: always` issues one on every token response. The signing key
is kept under your user config dir (`…/bohurupee/oidc.key`) so it survives
restarts.

Provider-shaped payloads: [`docs/provider-profiles.md`](docs/provider-profiles.md).
Adding a built-in template: [`CONTRIBUTING.md`](CONTRIBUTING.md).

Step-by-step curl: [`examples/curl/README.md`](examples/curl/README.md).
Any framework / Auth.js: [`docs/any-framework.md`](docs/any-framework.md).
Laravel Socialite: [`docs/socialite.md`](docs/socialite.md),
[`examples/laravel-socialite`](examples/laravel-socialite), and the sibling
package `milon/bohurupee-laravel`.
Dependency-free Python OIDC client:
[`examples/oidc-client`](examples/oidc-client/README.md).

## Dependencies

Codes and bearer tokens are random hex in process memory; they vanish when the
binary exits.

| Dependency                                       | Why                                                                                                                                                                               |
|--------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Go 1.25+** (toolchain)                         | One static binary, no Node/PHP required to run the IdP. Module path is `github.com/milon/bohurupee`. 1.24+ so darwin builds include `LC_UUID` (required by dyld on recent macOS). |
| **`gopkg.in/yaml.v3`**                           | Load personas, PKCE, id_token mode, and provider profiles.                                                                                                                        |
| **`github.com/lestrrat-go/jwx/v3`**              | Sign and serve RS256 `id_token` / JWKS.                                                                                                                                           |
| **`flag` (stdlib)**                              | Bind/port/config flags and a danger override.                                                                                                                                     |
| **`net/http` (stdlib)**                          | Method-aware patterns cover home + OAuth without a third-party router.                                                                                                            |
| **`html/template` + `embed` (stdlib)**           | Home page, consent UI, and `form_post` auto-submit HTML.                                                                                                                          |
| **`net` (stdlib)**                               | Loopback check via `net.ParseIP` / `IP.IsLoopback` (and DNS lookup for hostnames like `localhost`).                                                                               |
| **`crypto/rand` + `encoding/hex` (stdlib)**      | Authorization codes and access tokens.                                                                                                                                            |
| **`crypto/sha256` + `encoding/base64` (stdlib)** | PKCE S256.                                                                                                                                                                        |
| **`encoding/json` (stdlib)**                     | Token + userinfo JSON.                                                                                                                                                            |
| **`net/url` (stdlib)**                           | Redirect `code`/`state` query on `redirect_uri`.                                                                                                                                  |
| **`sync` / `time` (stdlib)**                     | In-memory grant store, 2 minute code TTL, 1 hour token TTL.                                                                                                                       |
| **GoReleaser**                                   | `.goreleaser.yaml` publishes darwin/linux/windows archives and a multi-arch scratch image on a `v*` tag. See [CONTRIBUTING.md](CONTRIBUTING.md) to add a response template.       |

## License

MIT
