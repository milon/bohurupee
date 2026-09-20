# Get started

## Install

### Homebrew

```bash
brew install --cask milon/bohurupee/bohurupee
bohurupee init
bohurupee
```

That taps [milon/homebrew-bohurupee](https://github.com/milon/homebrew-bohurupee).
macOS and Linux are both in the cask.

### Release binary

Download from the
[latest release](https://github.com/milon/bohurupee/releases/latest). Archives
are named `bohurupee_<version>_<os>_<arch>.tar.gz` (zip on Windows).

```bash
VERSION=0.2.1   # or newer — check the releases page
curl -fsSL -o bohurupee.tar.gz \
  "https://github.com/milon/bohurupee/releases/download/v${VERSION}/bohurupee_${VERSION}_darwin_arm64.tar.gz"
tar -xzf bohurupee.tar.gz
./bohurupee init
./bohurupee
```

Verify against `checksums.txt` from the same release before running a binary
you downloaded.

### Docker

The image is the static binary on scratch. Inside the container it listens on
`0.0.0.0` so Docker can publish the port. **Publish on loopback only:**

```bash
docker run --rm -p 127.0.0.1:4190:4190 ghcr.io/milon/bohurupee:v0.2.1
```

If you mount a config file, set `bind: 0.0.0.0` in it. `bind: 127.0.0.1`
listens on container loopback, which Docker cannot publish.

```bash
docker run --rm -v "$PWD:/work" -w /work ghcr.io/milon/bohurupee:v0.2.1 init
```

### From source

Requires Go 1.25+. See [Contributing](contributing.md).

```bash
git clone https://github.com/milon/bohurupee.git
cd bohurupee
go build -o ./bohurupee ./cmd/bohurupee
./bohurupee init
./bohurupee
```

## First sign-in

1. Open [http://127.0.0.1:4190](http://127.0.0.1:4190/) — the dashboard lists
   personas and copy-paste URLs.
2. Start an authorize request (example):

```text
http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http%3A%2F%2F127.0.0.1%3A9999%2Fcallback&response_type=code&state=xyz&scope=openid%20profile%20email
```

3. Pick a persona, or click **Deny** (`error=access_denied`).
4. Your app exchanges `code` at `POST /google/token`, then calls
   `GET /google/userinfo` with the access token.

!!! tip "Open client"
    Any `client_id` and `client_secret` are accepted by default. Secrets are
    not validated. See [`openClient` and `clients`](configuration.md#openclient)
    if you want an allowlist.

### Skip the consent page

| Mechanism | Use when |
|-----------|----------|
| `?auto=alice` | Scripts and curl |
| `BOHURUPEE_AUTO_APPROVE=1` | Every authorize uses the default persona |
| `POST /__login` | Browser tests (`loginAs`) without rewriting authorize URLs |

Playwright and PHP helpers:
[examples/playwright](https://github.com/milon/bohurupee/tree/master/examples/playwright),
[examples/php](https://github.com/milon/bohurupee/tree/master/examples/php).

### Curl / OIDC smoke tests

From a git checkout with the server already running:

```bash
./examples/curl/run-all.sh
python3 examples/oidc-client/client.py
```

## Configure personas

`bohurupee init` writes `bohurupee.yaml`. Edit people without rebuilding:

```yaml
personas:
  - id: alice
    email: alice@example.com
    name: Alice Admin
    claims:
      role: admin
  - id: bob
    email: bob@example.com
    name: Bob User
    email_verified: false
```

Apply changes while the server is up:

```bash
curl -s -X POST http://127.0.0.1:4190/__reload
```

Every YAML key is documented in [Configuration](configuration.md). Routes and
flags are in [Endpoints and CLI](endpoints.md).

## Wire your app

- Generic OIDC / Auth.js → [Any framework](any-framework.md)
- Laravel Socialite → [Laravel Socialite](socialite.md)
- GitHub-shaped userinfo → [Provider profiles](provider-profiles.md)
