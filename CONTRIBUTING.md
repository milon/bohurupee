# Contributing

Bohurupee is a local fake identity provider. Keep the default userinfo
generic. Provider-shaped JSON is opt-in.

## Develop

Go 1.25+ is required to build. Recent macOS dyld refuses binaries without a
Mach-O `LC_UUID`; the Go linker started emitting that in 1.24. The module
path is `github.com/milon/bohurupee`.

```bash
go test ./...
golangci-lint run
go build -o ./bohurupee ./cmd/bohurupee
```

CI runs `golangci-lint` and `go test` as separate jobs; either failing blocks
the PR. People running a release binary do not need this toolchain. See the
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
| **GoReleaser** | `.goreleaser.yaml` publishes darwin, linux, and windows archives, plus a multi-arch scratch image, on a `v*` tag. The release workflow then renders the Homebrew 7 cask from `dist/checksums.txt` (`scripts/render-homebrew-cask.sh`) and copies it into [milon/homebrew-bohurupee](https://github.com/milon/homebrew-bohurupee). |

## Release

1. Land changes on `master`. Update [CHANGELOG.md](CHANGELOG.md) (move
   Unreleased notes under a dated version).
2. Tag and push the binary repo, for example `v0.5.0`:

   ```bash
   git tag v0.5.0
   git push origin v0.5.0
   ```

   That runs tests, publishes GitHub Release assets, pushes
   `ghcr.io/milon/bohurupee:v0.5.0`, and updates the Homebrew cask (requires
   the `HOMEBREW_TAP_DEPLOY_KEY` secret).
3. Tag the same version on [milon/bohurupee-laravel](https://github.com/milon/bohurupee-laravel)
   so Packagist matches.
4. Point README install examples at the new version if they still name an
   older tag.

Do not reintroduce `ids:` on the GoReleaser `homebrew_casks` block; that
broke cask archive binding on `v0.2.0`. The tap cask is Homebrew 7 DSL
(`arch`/`os`/`sha256`/`postflight_steps`); generate it with
`scripts/render-homebrew-cask.sh` instead of copying GoReleaser's nested
`on_macos`/`postflight` output.

The GitHub Pages site is MkDocs Material. Content lives in `docs/`. See the
[Contributing](https://milon.github.io/bohurupee/contributing/) page on the
docs site for the same guide as `docs/contributing.md`.

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

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
