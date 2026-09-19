# Contributing

Bohurupee is a local fake identity provider. Keep the default userinfo
generic. Provider-shaped JSON is opt-in.

## Develop

Go 1.25+ is required to build. Recent macOS dyld refuses binaries without a
Mach-O `LC_UUID`; the Go linker started emitting that in 1.24. The module
path is `github.com/milon/bohurupee`.

```bash
go test ./...
go build -o ./bohurupee ./cmd/bohurupee
```

People running a release binary do not need this toolchain. See the
[README](README.md).

## Dependencies

Codes and bearer tokens are random hex in process memory. They vanish when
the binary exits.

| Dependency | Why |
|------------|-----|
| **Go 1.25+** | One static binary. No Node or PHP is required to run the IdP. |
| **`gopkg.in/yaml.v3`** | Load personas, PKCE, id_token mode, and provider profiles. |
| **`github.com/lestrrat-go/jwx/v3`** | Sign and serve RS256 `id_token` / JWKS. |
| **`flag`** (stdlib) | Bind, port, config, and the non-loopback override. |
| **`net/http`** (stdlib) | Home page and OAuth routes, without a third-party router. |
| **`html/template` + `embed`** (stdlib) | Home page, consent UI, and `form_post` auto-submit HTML. |
| **`net`** (stdlib) | Loopback check via `net.ParseIP` / `IP.IsLoopback`, including hostnames such as `localhost`. |
| **`crypto/rand` + `encoding/hex`** (stdlib) | Authorization codes and access tokens. |
| **`crypto/sha256` + `encoding/base64`** (stdlib) | PKCE S256. |
| **`encoding/json`** (stdlib) | Token and userinfo JSON. |
| **`net/url`** (stdlib) | `code` and `state` on `redirect_uri`. |
| **`sync` / `time`** (stdlib) | In-memory grant store. Codes last 2 minutes. Tokens last 1 hour. |
| **GoReleaser** | `.goreleaser.yaml` publishes darwin, linux, and windows archives, plus a multi-arch scratch image, on a `v*` tag. |

## Add a response template

Built-in shapes live in `internal/profiles/templates.go`. A template adds
fields on top of the generic persona. It does not replace `id`, `email`,
`name`, `nickname`, or `avatar`.

1. Add a function with the same signature as `githubTemplate`.
2. Register it in the `templates` map. The key is the YAML name (`github`,
   `google`, …) and must be lowercase.
3. Add a case in `internal/profiles/render_test.go`. Assert the new fields
   and that the generic getters are still present.
4. Add a row to the table in `docs/provider-profiles.md`.
5. Optionally add a `providerProfiles` entry in `bohurupee.example.yaml`.

Unknown `responseTemplate` values are rejected at startup.

You do not need a code change for a one-off shape. Set `response:` in
`bohurupee.yaml` instead. See `docs/provider-profiles.md`.
