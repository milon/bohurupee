---
title: Configuration
---

# Configuration (`bohurupee.yaml`)

Bohurupee loads YAML once at startup as a **sparse overlay** on the built-in
default config (same content as `bohurupee init` / `bohurupee.example.yaml`:
Alice, Bob, Carol, and the stock provider profiles). Omitted keys keep their
defaults, except **`personas`, which is required** in every config file.
With no `--config` flag it uses `./bohurupee.yaml` when that file exists;
otherwise it runs with those built-ins on port `4190`, bind `127.0.0.1`.

```bash
bohurupee init                  # write ./bohurupee.yaml
bohurupee                       # load ./bohurupee.yaml if present
bohurupee --config ./other.yaml
```

While the process is running, apply file changes without restart (loopback
only):

```bash
curl -s -X POST http://127.0.0.1:4190/__reload
```

Reload updates personas, PKCE, `idToken`, `openClient`, `refreshTokens`,
`clients`, and `providerProfiles`. It does **not** change the listen address
or OIDC signing key. Invalid YAML leaves the previous config in place and
returns an error JSON body.

CLI `--bind` / `--port` override the file for the current process only.

---

## Full example

This matches the shipped [`bohurupee.example.yaml`](https://github.com/milon/bohurupee/blob/master/bohurupee.example.yaml), with optional keys shown in comments:

```yaml
port: 4190
bind: 127.0.0.1
pkce: optional          # optional | required | forbidden
idToken: openid         # openid | always
openClient: true
# refreshTokens: false  # default; set true to issue refresh_token
# clients:
#   - id: my-app
#     redirect_uris:
#       - http://127.0.0.1:3000/api/auth/callback/bohurupee

personas:
  - id: alice
    email: alice@example.com
    name: Alice Admin
    nickname: alice
    claims:
      role: admin
  - id: bob
    email: bob@example.com
    name: Bob User
    email_verified: false
    claims:
      role: user
  - id: carol
    email: carol@example.com
    name: Carol Reviewer
    nickname: carol
    avatar: ""
    response:
      picture: ""

providerProfiles:
  github:
    responseTemplate: github
  google:
    responseTemplate: google
  facebook:
    responseTemplate: facebook
    endpoints:
      userinfo: /me
  apple:
    responseTemplate: apple
    protocol:
      response_mode: form_post
      id_token: true
```

---

## Top-level keys

### `port`

| | |
|--|--|
| **Type** | integer |
| **Default** | `4190` |
| **Reload** | ignored (listen address fixed at startup) |

TCP port to listen on. `0` in the file falls back to the CLI default.

```yaml
port: 4190
```

Equivalent flag: `--port 4190`.

---

### `bind`

| | |
|--|--|
| **Type** | string |
| **Default** | `127.0.0.1` |
| **Reload** | ignored |

Host interface to bind. Non-loopback addresses are refused unless you pass
`--dangerously-bind-all-interfaces` (or run the Docker image, which binds
`0.0.0.0` inside the container on purpose).

In Docker, a loopback `bind` from the YAML (`127.0.0.1`, `localhost`, `::1`)
is upgraded to `0.0.0.0` so `-p` publishing works. The rest of the config
still loads. An explicit `--bind` always wins.

```yaml
bind: 127.0.0.1
```

```yaml
# only with --dangerously-bind-all-interfaces (outside Docker)
bind: 0.0.0.0
```

Equivalent flag: `--bind 127.0.0.1`.

When `--config` is omitted, Bohurupee looks for `./bohurupee.yaml`. In Docker
it also tries `/bohurupee.yaml` and `/config/bohurupee.yaml`, or
`BOHURUPEE_CONFIG` if set.

---

### `pkce`

| | |
|--|--|
| **Type** | string |
| **Default** | `optional` |
| **Values** | `optional`, `required`, `forbidden` |
| **Reload** | yes |

Controls Proof Key for Code Exchange on authorize / token.

| Value | Authorize | Token |
|-------|-----------|-------|
| `optional` | Challenge allowed but not required | Verifier checked when a challenge was sent |
| `required` | `code_challenge` must be present | Matching `code_verifier` required |
| `forbidden` | Challenge rejected | — |

```yaml
pkce: optional
```

```yaml
pkce: required   # force S256 (or plain) on every login
```

```yaml
pkce: forbidden  # reject clients that send a challenge
```

Supported challenge methods: `S256` and `plain`.

---

### `idToken`

| | |
|--|--|
| **Type** | string |
| **Default** | `openid` |
| **Values** | `openid`, `always` |
| **Reload** | yes |

When the token response includes an RS256 `id_token`.

| Value | Behavior |
|-------|----------|
| `openid` | Issue `id_token` when the authorize `scope` contains `openid` |
| `always` | Always issue `id_token`, even without `openid` |

A provider profile may also force tokens with `protocol.id_token: true`
(Apple in the example config).

```yaml
idToken: openid
```

```yaml
idToken: always
```

Claims on the JWT include `iss`, `sub`, `aud`, `iat`, `exp`, `email`,
`email_verified`, `name`, `given_name`, `family_name`, `nickname`, `picture`,
optional `nonce`, `at_hash`, and non-reserved keys from persona `claims`.

---

### `openClient`

| | |
|--|--|
| **Type** | boolean |
| **Default** | `true` |
| **Reload** | yes |

Open local client mode.

| Value | Behavior |
|-------|----------|
| `true` | Any `client_id` works. If that id is listed under `clients` **with** `redirect_uris`, those URIs are enforced. Unlisted clients accept any absolute `http`/`https` redirect. |
| `false` | Every `client_id` must appear under `clients`. Clients with `redirect_uris` must use one of those URIs. |

```yaml
openClient: true
```

```yaml
openClient: false
clients:
  - id: strict-app
    redirect_uris:
      - http://127.0.0.1:3000/callback
```

Secrets are never validated either way.

---

### `refreshTokens`

| | |
|--|--|
| **Type** | boolean |
| **Default** | `false` |
| **Reload** | yes |

Opt-in refresh tokens. When `false` (default), the token JSON has no
`refresh_token` field and discovery lists only `authorization_code`.

When `true`:

- Code exchange may return `refresh_token`
- Discovery adds `refresh_token` to `grant_types_supported`
- `grant_type=refresh_token` issues a new access token (same refresh token reused)

```yaml
refreshTokens: false
```

```yaml
refreshTokens: true
```

Access tokens last 1 hour; refresh tokens last 24 hours (in memory).

---

## `personas`

| | |
|--|--|
| **Type** | list of objects |
| **Default** | Alice, Bob, Carol when no config file is loaded |
| **Reload** | yes |

**Required** in every config file. An omitted `personas` key is an error (the
built-in list is not inherited). An empty list is also an error. Each persona
is a fake user you can pick on the consent page or select with `?auto=<id>`.

### Persona fields

#### `id` (required)

URL-safe slug: letters, digits; after the first character also `-` and `_`.
Used in consent, `auto=`, and stable subject ids (`{provider}:{id}`).

```yaml
- id: alice
```

#### `email`

Email claim on userinfo and `id_token`.

```yaml
  email: alice@example.com
```

#### `email_verified`

| | |
|--|--|
| **Type** | boolean |
| **Default** | `true` when omitted |

```yaml
  email_verified: false
```

#### `name`

Display name. Split into `given_name` / `family_name` on templates and
`id_token` (“Alice Admin” → Alice / Admin).

```yaml
  name: Alice Admin
```

#### `nickname`

Short handle. Defaults to `id` when omitted. Often mapped to Socialite
`getNickname()` / GitHub `login`.

```yaml
  nickname: alice
```

#### `avatar`

Profile image URL.

| Omission | Result |
|----------|--------|
| Key omitted | Dicebear identicon: `https://api.dicebear.com/9.x/identicon/svg?seed={id}` |
| `avatar: ""` | Empty string (missing picture tests) |
| Non-empty string | That URL |

```yaml
  avatar: https://example.com/alice.png
```

```yaml
  avatar: ""
```

#### `claims`

Arbitrary JSON object merged into userinfo and `id_token` (except reserved
JWT keys such as `iss`, `sub`, `aud`, `exp`, `iat`, `nonce`, `at_hash`).

```yaml
  claims:
    role: admin
    org_id: "42"
```

#### `response`

Overlay map applied **last** in the userinfo merge (can clear getters).
String values may use placeholders: `{{id}}`, `{{sub}}`, `{{email}}`,
`{{name}}`, `{{nickname}}`, `{{avatar}}`, `{{provider}}`.

```yaml
  response:
    picture: ""
    title: "{{nickname}}"
```

### Persona examples

Admin with role claim:

```yaml
personas:
  - id: alice
    email: alice@example.com
    name: Alice Admin
    nickname: alice
    claims:
      role: admin
```

Unverified email:

```yaml
  - id: bob
    email: bob@example.com
    name: Bob User
    email_verified: false
    claims:
      role: user
```

No avatar (Socialite / UI empty-picture cases):

```yaml
  - id: carol
    email: carol@example.com
    name: Carol Reviewer
    nickname: carol
    avatar: ""
    response:
      picture: ""
```

---

## `clients`

| | |
|--|--|
| **Type** | list of objects |
| **Default** | none (open client) |
| **Reload** | yes |

Optional registered OAuth clients. Duplicate `id` values are rejected.

### Client fields

#### `id` (required)

Must match the authorize / token `client_id`.

#### `redirect_uris`

Exact-match allowlist. When non-empty, authorize rejects any other
`redirect_uri` for this client (HTTP 400, no redirect).

```yaml
clients:
  - id: my-app
    redirect_uris:
      - http://127.0.0.1:3000/api/auth/callback/bohurupee
      - http://localhost:3000/api/auth/callback/bohurupee
```

With `openClient: true`, clients **not** listed here still accept any valid
redirect. With `openClient: false`, unlisted `client_id` values are rejected.

---

## `providerProfiles`

| | |
|--|--|
| **Type** | map keyed by provider slug |
| **Default** | stock profiles from `bohurupee.example.yaml` (Apple `form_post`, Facebook `/me`, …) |
| **Reload** | yes |

Per-slug userinfo shape, extra routes, and protocol defaults. Key must be a
valid provider slug (same rules as persona `id`). Entries in your file
**merge** into the stock profile for that slug; omitted slugs stay as
defaults.

See [Provider profiles](provider-profiles.html) for merge order and the full
template table. Summary of profile fields:

### `responseTemplate`

Built-in template name: `github`, `google`, `facebook`, `generic`, …  
Unknown names are rejected at load / reload.

```yaml
providerProfiles:
  github:
    responseTemplate: github
```

### `response`

Custom JSON merged into userinfo (supports `{{placeholders}}`).

```yaml
  staffdir:
    response:
      title: Engineer
      username: "{{nickname}}"
```

### `endpoints`

Extra paths. Today `userinfo` registers an alias on that provider:

```yaml
  facebook:
    responseTemplate: facebook
    endpoints:
      userinfo: /me    # also GET /facebook/me
```

:::note
Userinfo alias routes are registered when the process starts. Adding a new
alias via reload updates rendering for existing routes; a **new** path may
need a process restart to appear on the mux.
:::

### `protocol`

Defaults when the authorize query omits the matching parameter:

| Key | Values | Effect |
|-----|--------|--------|
| `response_mode` | `query` (default), `form_post` | How `code` / `error` return to the app |
| `id_token` | `true` / `false` | Force `id_token` on token response |

```yaml
  apple:
    responseTemplate: apple
    protocol:
      response_mode: form_post
      id_token: true
```

The client can still pass `response_mode=query` explicitly.

---

## What is not in the YAML

| Concern | Where it lives |
|---------|----------------|
| Listen override | `--bind`, `--port`, `--dangerously-bind-all-interfaces` |
| Auto-approve every authorize | env `BOHURUPEE_AUTO_APPROVE=1` |
| OIDC signing key | `~/…/bohurupee/oidc.key` (created on first run) |
| Codes / tokens | Process memory only |

---

## Minimal configs

No file — built-in Alice / Bob / Carol on `127.0.0.1:4190`:

```bash
bohurupee
```

A config file must include `personas`. Other keys stay at the defaults:

```yaml
pkce: required
port: 5190
personas:
  - id: alice
    email: alice@example.com
    name: Alice
  - id: bob
    email: bob@example.com
    name: Bob
    email_verified: false
```

Partial provider profile (merges into the stock entry):

```yaml
personas:
  - id: alice
    email: alice@example.com
    name: Alice
providerProfiles:
  apple:
    protocol:
      response_mode: query
```

Strict local app:

```yaml
openClient: false
clients:
  - id: next-local
    redirect_uris:
      - http://localhost:3000/api/auth/callback/bohurupee
personas:
  - id: alice
    email: alice@example.com
    name: Alice Admin
```
