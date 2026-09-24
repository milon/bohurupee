# Contributing

Bohurupee is a local fake identity provider. Keep the default userinfo
**generic**. Provider-shaped JSON is opt-in via templates and YAML.

## Develop

Go **1.25+** is required (module path `github.com/milon/bohurupee`).

```bash
git clone https://github.com/milon/bohurupee.git
cd bohurupee
go test ./...
golangci-lint run
go build -o ./bohurupee ./cmd/bohurupee
./bohurupee --config ./bohurupee.example.yaml
```

CI runs **lint** and **tests** as separate jobs — either failing blocks the
PR. You do not need Go to *run* a release binary.

### Layout

| Path | Role |
|------|------|
| `cmd/bohurupee` | CLI entrypoint |
| `internal/server` | HTTP handlers, CORS, reload |
| `internal/oauth` | Codes, tokens, PKCE, personas |
| `internal/oidc` | Discovery, JWKS, `id_token` |
| `internal/profiles` | Userinfo templates and merge |
| `internal/config` | YAML loading |
| `internal/ui` | Embedded HTML/CSS |
| `docs/` | This documentation site |
| `examples/` | curl, Auth.js, Laravel, Playwright, … |

## Dependencies (why they exist)

| Dependency | Why |
|------------|-----|
| Go 1.25+ | One static binary |
| `gopkg.in/yaml.v3` | Config |
| `github.com/lestrrat-go/jwx/v3` | RS256 `id_token` / JWKS |
| stdlib `net/http`, `html/template`, `embed` | Server and UI |
| GoReleaser | Multi-arch binaries, Docker image, Homebrew cask render |

Codes and bearer tokens are random hex in memory. They vanish when the
process exits.

## Add a response template

Built-in shapes live in `internal/profiles/templates.go`. A template **adds**
fields on top of the generic persona. It does not replace `id`, `email`,
`name`, `nickname`, or `avatar`.

1. Add a function with the same signature as `githubTemplate`.
2. Register it in the `templates` map (lowercase YAML key).
3. Add assertions in `internal/profiles/render_test.go` (new fields **and**
   generic getters still present).
4. Document a row in [Provider profiles](provider-profiles.md).
5. Optionally add a `providerProfiles` entry in `bohurupee.example.yaml`.

Unknown `responseTemplate` values are rejected at startup / reload.

For a one-off shape, prefer YAML `response:` — no Go change required. See
[Configuration](configuration.md) and [Provider profiles](provider-profiles.md).

## Docs site

Content is MkDocs Material under `docs/`.

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

Open [http://127.0.0.1:8000](http://127.0.0.1:8000/). Pushes to `master` that
touch `docs/` or `mkdocs.yml` deploy via GitHub Pages
(`.github/workflows/docs.yml`). Enable **Settings → Pages → GitHub Actions**
once if needed.

Keep prose concrete and short. Prefer tables for option lists. Link to
working examples under `examples/` with full GitHub URLs.

## Pull requests

- Match surrounding code style; run `gofmt`, `go test ./...`, and
  `golangci-lint run` before opening a PR.
- Do not expand the template list unless a real client is missing a field.
- Do not add framework adapters casually — Auth.js helper and Laravel
  Socialite already cover the intended surface.
- Never commit secrets, local `bohurupee.yaml` overrides, or signing keys.

## Release (maintainers)

1. Land changes on `master`. Update `CHANGELOG.md`.
2. Tag and push the binary repo, e.g. `v0.5.0` — GoReleaser publishes
   archives, GHCR image, and the release workflow updates
   [homebrew-bohurupee](https://github.com/milon/homebrew-bohurupee)
   (needs `HOMEBREW_TAP_DEPLOY_KEY`).
3. Tag the same version on
   [milon/bohurupee-laravel](https://github.com/milon/bohurupee-laravel)
   so Packagist matches.
4. Refresh README install version pins if they still name an older tag.

Do **not** reintroduce `ids:` on the GoReleaser `homebrew_casks` block — that
broke cask archive binding on `v0.2.0`. The tap file is Homebrew 7 DSL
(`arch`/`os`/`sha256`/`postflight_steps`); keep generating it with
`scripts/render-homebrew-cask.sh` rather than copying GoReleaser's nested
`on_macos`/`postflight` output.

## Security reports

Bohurupee is local / DEV ONLY. File ordinary
[GitHub issues](https://github.com/milon/bohurupee/issues) for bugs. See
[Security](security.md).
