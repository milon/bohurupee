# Endpoints and CLI

Bohurupee speaks the OAuth 2.0 authorization-code flow and OpenID Connect
discovery. Every `{provider}` is a URL slug (`google`, `github`, `acme`, …).
The same persona through different slugs gets different subjects
(`google:alice` vs `acme:alice`).

Base URL (default): `http://127.0.0.1:4190`

---

## HTTP routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/` | Dashboard (personas, copy-paste URLs) |
| `GET` | `/{provider}/authorize` | Consent / auto-approve / deny |
| `POST` | `/{provider}/token` | Code or refresh → tokens |
| `GET` | `/{provider}/userinfo` | Bearer profile JSON |
| `GET` | `/{provider}/.well-known/openid-configuration` | OIDC discovery |
| `GET` | `/{provider}/jwks` | JWKS |
| `GET` | `/{provider}/auth/keys` | JWKS alias |
| `GET`/`POST` | `/__login` | Test helper: set `loginAs` cookie |
| `POST` | `/__reload` | Reload YAML (loopback only) |

Provider profiles may add aliases (for example `GET /facebook/me`).

CORS (loopback `Origin` only) is enabled on discovery, token, and userinfo so
a browser app on another localhost port can finish the code flow.

---

## `GET /{provider}/authorize`

Starts login. On success Bohurupee redirects (or form-posts) to `redirect_uri`
with `code` and `state`. On user cancel or validation errors with a valid
redirect, it returns `error` / `error_description` (RFC 6749).

### Query parameters

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | yes | Any string when `openClient` is true |
| `redirect_uri` | yes | Absolute `http` or `https` URL, no fragment |
| `response_type` | yes | Must be `code` |
| `state` | yes | Opaque value echoed back |
| `scope` | no | Space-separated; include `openid` for `id_token` (unless `idToken: always` or profile forces it) |
| `nonce` | no | Copied into `id_token` when present |
| `response_mode` | no | `query` (default) or `form_post` |
| `code_challenge` | depends on `pkce` | PKCE challenge |
| `code_challenge_method` | no | `S256` (preferred) or `plain` |
| `auto` | no | Persona id — skip consent and issue a code |
| `deny` | no | Truthy (`1`, `true`, …) — `error=access_denied` |
| `prompt` | no | `login` or `select_account` forces the consent UI (ignores last-persona and `loginAs` cookies; `auto=` still wins) |
| `login_hint` | no | Persona id to highlight on the consent page |

### Example

```text
http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http%3A%2F%2F127.0.0.1%3A9999%2Fcallback&response_type=code&state=xyz&scope=openid%20profile%20email
```

Auto-approve Alice:

```text
…&auto=alice
```

---

## `POST /{provider}/token`

`Content-Type: application/x-www-form-urlencoded`.

Client auth: HTTP Basic **or** `client_id` / `client_secret` form fields
(secret accepted, not checked).

### Authorization code grant

| Field | Required |
|-------|----------|
| `grant_type` | `authorization_code` |
| `code` | yes |
| `redirect_uri` | yes (must match authorize) |
| `client_id` | yes (unless Basic) |
| `code_verifier` | when PKCE was used |

Success (default, no refresh):

```json
{
  "access_token": "…",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "openid profile email",
  "id_token": "…"
}
```

With `refreshTokens: true`, a `refresh_token` field is added.

Errors are JSON: `{"error":"invalid_grant","error_description":"…"}`.

### Refresh grant (opt-in)

Requires `refreshTokens: true` in YAML.

| Field | Required |
|-------|----------|
| `grant_type` | `refresh_token` |
| `refresh_token` | yes |
| `client_id` | yes |

---

## `GET /{provider}/userinfo`

```http
Authorization: Bearer <access_token>
```

Returns the merged persona / profile JSON. Generic shape:

```json
{
  "id": "google:alice",
  "sub": "google:alice",
  "email": "alice@example.com",
  "email_verified": true,
  "name": "Alice Admin",
  "nickname": "alice",
  "avatar": "https://api.dicebear.com/9.x/identicon/svg?seed=alice"
}
```

Provider templates add fields (`login`, `avatar_url`, …). See
[Provider profiles](provider-profiles.md).

---

## Discovery and JWKS

```text
GET /{provider}/.well-known/openid-configuration
GET /{provider}/jwks
```

Issuer is `http://127.0.0.1:4190/{provider}` (or whatever host you bound).
Discovery lists authorize, token, userinfo, jwks, supported grants, PKCE
methods, and `claims_supported`.

---

## `POST /__login`

Test helper used by Playwright / PHP `loginAs`. Sets an HttpOnly cookie so the
**next** authorize in that cookie jar skips consent.

| Field | Required | Notes |
|-------|----------|-------|
| `persona` | yes | Must exist in config |
| `provider` | no | Echoed in the JSON response |

```bash
curl -s -c cookies.txt -X POST http://127.0.0.1:4190/__login \
  -d 'persona=alice&provider=google'
```

---

## `POST /__reload`

Loopback only (`127.0.0.1`, `::1`). Reloads the YAML path used at startup.

```bash
curl -s -X POST http://127.0.0.1:4190/__reload
```

Example success body:

```json
{
  "ok": true,
  "config": "bohurupee.yaml",
  "personas": ["alice", "bob", "carol"],
  "refreshTokens": false
}
```

---

## CLI

```bash
bohurupee [flags]
bohurupee init [--config path] [--force]
bohurupee --version
```

| Flag | Default | Purpose |
|------|---------|---------|
| `--config` | `./bohurupee.yaml` if present | YAML path |
| `--bind` | `127.0.0.1` (or file) | Listen host |
| `--port` | `4190` (or file) | Listen port |
| `--dangerously-bind-all-interfaces` | off | Allow non-loopback bind |
| `--version` | | Print version and exit |

### `bohurupee init`

Writes a starter config (same shape as `bohurupee.example.yaml`). Refuses to
overwrite unless `--force`.

```bash
bohurupee init
bohurupee init --config ./dev.yaml
bohurupee init --force
```

### Environment

| Variable | Effect |
|----------|--------|
| `BOHURUPEE_AUTO_APPROVE` | Truthy (`1`, `true`, `yes`, `alice`) — every authorize auto-picks the default persona (unless `prompt=login` or `auto=` / deny) |
| `BOHURUPEE_IN_DOCKER` | Set by the official image; allows `0.0.0.0` bind inside the container; upgrades loopback `bind` from YAML |
| `BOHURUPEE_CONFIG` | Absolute/relative path to YAML when `--config` is omitted |

---

## Lifetimes

| Item | TTL |
|------|-----|
| Authorization code | 2 minutes, single use |
| Access token | 1 hour |
| Refresh token | 24 hours (when enabled) |

All grants are in-process memory and disappear when the binary exits.
