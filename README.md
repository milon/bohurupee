<img src="assets/logo.svg" alt="" width="88" align="right">

# Bohurupee (বহুরূপী)

Local fake identity provider for social login development.

Run it on your machine and point any OAuth or OpenID Connect client at
`http://127.0.0.1:4190`. The client thinks it is talking to Google, GitHub,
Apple, or a provider you invent. Nothing leaves localhost.

Laravel Socialite is an optional adapter (`milon/bohurupee-laravel`). You do
not need it, or Go, to use the server.

**DEV ONLY.** v0.1 listens on loopback unless you override that on purpose.

## Install

Download a release binary, install with Homebrew, run the container, or build
from source. The server is one file. It does not need Node, PHP, or a database.

### Homebrew

```bash
brew install --cask milon/bohurupee/bohurupee
bohurupee init
bohurupee
```

That taps [milon/homebrew-bohurupee](https://github.com/milon/homebrew-bohurupee).
macOS and Linux are both in the cask.

### Release binary

[v0.1](https://github.com/milon/bohurupee/releases/tag/v0.1) ships
`bohurupee`, `bohurupee.example.yaml`, and `checksums.txt`.

| OS      | Architecture  | File                                |
|---------|---------------|-------------------------------------|
| macOS   | Apple silicon | `bohurupee_0.1_darwin_arm64.tar.gz` |
| macOS   | Intel         | `bohurupee_0.1_darwin_amd64.tar.gz` |
| Linux   | x86_64        | `bohurupee_0.1_linux_amd64.tar.gz`  |
| Linux   | arm64         | `bohurupee_0.1_linux_arm64.tar.gz`  |
| Windows | x86_64        | `bohurupee_0.1_windows_amd64.zip`   |

```bash
# example: macOS Apple silicon. Check checksums.txt from the same release first.
curl -fsSL -o bohurupee.tar.gz \
  https://github.com/milon/bohurupee/releases/download/v0.1/bohurupee_0.1_darwin_arm64.tar.gz
tar -xzf bohurupee.tar.gz
./bohurupee init
./bohurupee
```

`init` writes `bohurupee.yaml` in the current directory (Alice, Bob, Carol, and
the built-in provider profiles). It will not replace a file that is already
there unless you pass `--force`. Then start the server with no flags: it
loads `./bohurupee.yaml` on its own.

Open [http://127.0.0.1:4190/](http://127.0.0.1:4190/). The home page lists
personas, endpoints, and URLs you can copy.

### Docker

The image is the static binary on scratch. It has no shell. Inside the
container the process listens on `0.0.0.0` so Docker can publish the port.
Publish that port on loopback only:

```bash
docker run --rm -p 127.0.0.1:4190:4190 ghcr.io/milon/bohurupee:v0.1
```

Do not map `0.0.0.0:4190` on a shared network. If you mount a config file,
set `bind: 0.0.0.0` in it. `bind: 127.0.0.1` listens on container loopback,
which Docker cannot publish. To write that file on the host:

```bash
docker run --rm -v "$PWD:/work" -w /work ghcr.io/milon/bohurupee:v0.1 init
```

### From source

Requires Go 1.25+. See [CONTRIBUTING.md](CONTRIBUTING.md) for the toolchain
and library list.

```bash
go build -o ./bohurupee ./cmd/bohurupee
./bohurupee init
./bohurupee
```

## Use it

Bohurupee speaks the authorization-code flow. Pick a **provider slug** — any
URL-safe name — and treat `http://127.0.0.1:4190/{provider}` as the issuer.

`google`, `github`, and `acme` are all valid. The slug is only a namespace.
Alice signed in through `/google` is `google:alice`. The same person through
`/acme` is `acme:alice`.

### Point a client at it

| Setting       | Value                                                           |
|---------------|-----------------------------------------------------------------|
| Issuer        | `http://127.0.0.1:4190/google` (or any other slug)              |
| Discovery     | `http://127.0.0.1:4190/google/.well-known/openid-configuration` |
| Authorize     | `GET /google/authorize`                                         |
| Token         | `POST /google/token`                                            |
| Userinfo      | `GET /google/userinfo`                                          |
| JWKS          | `GET /google/jwks`                                              |
| Client ID     | any string, for example `dev-client`                            |
| Client secret | any string. It is accepted and not checked                      |
| Redirect URI  | your app's callback. Send the same value to authorize and token |
| Scopes        | `openid profile email` if you want an `id_token`                |

Authorization-code clients that cannot do discovery can hard-code the three
endpoints above. OIDC clients should use the discovery document. A worked
example for Auth.js and other stacks is in
[`docs/any-framework.md`](docs/any-framework.md).

### Sign in from a browser

Start the server, then open an authorize URL. The consent page lists the
personas from your config. Click one. Bohurupee redirects to `redirect_uri`
with `?code=` and the `state` you sent.

```text
http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http%3A%2F%2F127.0.0.1%3A9999%2Fcallback&response_type=code&state=xyz
```

The home page has this URL ready to copy, filled in with the address the
process is actually listening on.

Your app then `POST`s the code to `/google/token` and calls `/google/userinfo`
with the access token. Codes are single-use and last 2 minutes. Access tokens
last 1 hour. Both live in memory and disappear when the process exits.

### Skip the consent page

For curl, tests, and scripts, add `?auto=alice` (or `bob`, `carol`, …). The
server redirects immediately as that persona. `BOHURUPEE_AUTO_APPROVE=1`
does the same for every authorize request.

```bash
curl -sI 'http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http://127.0.0.1:9999/callback&response_type=code&state=xyz&auto=alice'
```

The callback host does not need to be running. Read `code` from the
`Location` header. Step-by-step token and userinfo calls are in
[`examples/curl/README.md`](examples/curl/README.md). From a git checkout:

```bash
./examples/curl/run-all.sh
python3 examples/oidc-client/client.py
```

The first script returns generic userinfo for `/acme`. The second returns
GitHub-shaped userinfo for `/github`. The Python client discovers the issuer,
uses PKCE, and verifies the `id_token`.

### What you get back

With no provider profile, userinfo is the same shape for every slug:

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

`id` and `sub` are `{provider}:{persona}`. Request `scope` containing
`openid` and the token response also includes an RS256 `id_token`. Verify it
against `/{provider}/jwks` (alias `/{provider}/auth/keys`). The signing key
is stored under your user config directory (`…/bohurupee/oidc.key`) so it
survives restarts.

### Personas and config

From the project directory:

```bash
bohurupee init
```

That writes `bohurupee.yaml`. Edit it, then start the server. It reads the
file on startup. You do not rebuild to change people. `--config` writes or
loads a different path. `--force` replaces a config that is already there.

If you would rather not run `init`, copy `bohurupee.example.yaml` to
`bohurupee.yaml`. The two files start out the same.

```yaml
port: 4190
bind: 127.0.0.1
pkce: optional          # optional | required | forbidden
idToken: openid         # openid (only when scope has openid) | always

personas:
  - id: alice
    email: alice@example.com
    name: Alice Admin
    nickname: alice
  - id: bob
    email: bob@example.com
    name: Bob User

providerProfiles:
  github:
    responseTemplate: github
```

`--bind` and `--port` on the command line override the file. If
`./bohurupee.yaml` exists and you omit `--config`, that file is loaded.

PKCE is optional by default. Set `pkce: required` when you want every
authorize request to send a `code_challenge`. The token request must then
include the matching `code_verifier`.

### Provider-shaped userinfo

A slug that matches a built-in template uses that shape. `/github/userinfo`
adds `login`, `avatar_url`, `html_url`, and `type` with no config. Any other
slug, such as `/acme` or a Socialite driver that has no template, uses the
default template: the generic fields plus `login`, `username`,
`preferred_username`, `display_name`, `picture`, `avatar_url`, and
`profile_image_url`.

Set `responseTemplate: generic` on a profile to return only the generic
fields. The example config also sets protocol and endpoints for a few slugs
(`facebook` `/me`, Apple `form_post`).

Generic fields (`id`, `email`, `name`, `nickname`, `avatar`) stay present so
client getters keep working.

Apple's profile defaults to `response_mode=form_post` and always returns an
`id_token`. Facebook's profile also serves userinfo at `/facebook/me`.

Custom keys without a built-in template:

```yaml
providerProfiles:
  staffdir:
    response:
      title: Engineer
      username: "{{nickname}}"
```

Merge order, templates, and placeholders are in
[`docs/provider-profiles.md`](docs/provider-profiles.md).

### Laravel Socialite

Optional. Install `milon/bohurupee-laravel`, set `BOHURUPEE_ENABLED=true` and
`BOHURUPEE_URL=http://127.0.0.1:4190`, and keep using
`Socialite::driver('google')`. The driver name is the provider slug.
Setup, the production guard, and a minimal app are in
[`docs/socialite.md`](docs/socialite.md) and
[`examples/laravel-socialite`](examples/laravel-socialite).

## Commands

```bash
bohurupee init
bohurupee init --config ./other.yaml
bohurupee init --force
```

`init` writes a starter `bohurupee.yaml` in the current directory. Pass
`--config` to choose another path. `--force` replaces a file that already
exists.

## Flags

| Flag                                | Default                       | Purpose                              |
|-------------------------------------|-------------------------------|--------------------------------------|
| `--bind`                            | `127.0.0.1`                   | Listen host                          |
| `--port`                            | `4190`                        | Listen port                          |
| `--config`                          | `./bohurupee.yaml` if present | Personas, PKCE, profiles, bind, port |
| `--version`                         |                               | Print the version and exit           |
| `--dangerously-bind-all-interfaces` | off                           | Allow a non-loopback bind            |

`0.0.0.0`, LAN addresses, and other non-loopback hosts are refused unless
that last flag is set. The Docker image sets `BOHURUPEE_IN_DOCKER=1`, which
is the same override inside the container.

## Endpoints

| Method | Path                                           | Notes                                                                                                                                                    |
|--------|------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GET`  | `/{provider}/authorize`                        | `client_id`, `redirect_uri`, `response_type=code`, `state`. Consent page, or `?auto=<persona>`. `response_mode=form_post` auto-posts `code` and `state`. |
| `POST` | `/{provider}/token`                            | Form body and/or HTTP Basic. `grant_type=authorization_code`. Send `code_verifier` when a PKCE challenge was used.                                       |
| `GET`  | `/{provider}/userinfo`                         | `Authorization: Bearer …`                                                                                                                                |
| `GET`  | `/{provider}/.well-known/openid-configuration` | Issuer, authorize, token, userinfo, jwks                                                                                                                 |
| `GET`  | `/{provider}/jwks`                             | JWKS. Alias: `/{provider}/auth/keys`                                                                                                                     |

## Security

This is a fake identity provider for development on your machine.

- Non-loopback binds are refused unless you pass `--dangerously-bind-all-interfaces`.
- In Docker, keep the published host port on `127.0.0.1`.
- Any `client_id` and `client_secret` are accepted.
- Codes and tokens are process memory. They are gone when the process exits.
- Do not point a production app at it, and do not expose it on a LAN or the public internet.

## License

MIT
