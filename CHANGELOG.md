# Changelog

All notable changes to Bohurupee are documented here.

## [0.5.0] - Unreleased

Post-v0.1 daily DX and protocol polish. Tag `v0.5.0` on this repo and
`milon/bohurupee-laravel` together so the binary, Docker image, Homebrew cask,
and Packagist package agree.

### Added
- Persona overlays (`email_verified`, `claims`, per-persona `response`)
- RFC-shaped OAuth errors; consent **Deny** → `error=access_denied`
- `POST /__login` plus Playwright and PHP `loginAs` helpers
- `POST /__reload` (loopback) to apply YAML without restart
- Loopback CORS on discovery, token, and userinfo
- Auth.js drop-in provider (`examples/authjs`)
- Optional refresh tokens (`refreshTokens: true`)
- `prompt=login` and `login_hint` on authorize
- Discovery `claims_supported`
- `id_token` `given_name` / `family_name` / `at_hash`
- Optional per-client `redirect_uris` (`clients` in YAML)
- `SECURITY.md`, GitHub issue templates, and `golangci-lint` in CI

### Changed
- Documentation site builds with [milon/papyrus](https://github.com/milon/papyrus) 1.5.0 (`papyrus.phar`; no Composer dep)

### Fixed
- Homebrew cask generation binds to release archives so a `v*` tag updates the tap
- Homebrew tap uses Homebrew 7 cask syntax (`arch`/`os`/`postflight_steps`)

## [0.2.1] - 2026-09-19

### Fixed
- Homebrew cask publishing from the release workflow (deploy key + archive binding)

## [0.2.0] - 2026-09-19

### Added
- Continued v0.2 line after 0.1.x (see GitHub compare for details)

## [0.1.5] - 2026-09

### Added
- Early post-v0.1 packaging fixes

## [0.1] - 2026-09

### Added
- Initial release: OAuth/OIDC local IdP, consent UI, personas, provider
  profiles, Docker image, and Homebrew cask
