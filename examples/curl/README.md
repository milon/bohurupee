# Curl against Bohurupee (generic OAuth)

These steps complete authorization-code flow as the hardcoded **Alice** persona.
There is no consent UI yet — auto-approve with `?auto=alice` or
`BOHURUPEE_AUTO_APPROVE=1`.

Any provider slug works (`google`, `acme`, …). The `id` / `sub` prefix follows
the path.

Start the server:

```bash
go run ./cmd/bohurupee
```

## 1. Authorize

```bash
curl -sI 'http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http://127.0.0.1:9999/callback&response_type=code&state=xyz&auto=alice'
```

Look for `Location: http://127.0.0.1:9999/callback?code=...&state=xyz`. Copy the
`code` query value. (`state` is echoed back.)

Equivalent with env auto-approve (no `auto=` query):

```bash
BOHURUPEE_AUTO_APPROVE=1 go run ./cmd/bohurupee
curl -sI 'http://127.0.0.1:4190/google/authorize?client_id=dev-client&redirect_uri=http://127.0.0.1:9999/callback&response_type=code&state=xyz'
```

## 2. Token

Open client: any `client_secret` is accepted. Body credentials or HTTP Basic.

```bash
curl -s -X POST 'http://127.0.0.1:4190/google/token' \
  -u 'dev-client:any-secret' \
  -d 'grant_type=authorization_code' \
  -d 'code=PASTE_CODE' \
  -d 'redirect_uri=http://127.0.0.1:9999/callback'
```

Or all in the form body:

```bash
curl -s -X POST 'http://127.0.0.1:4190/google/token' \
  -d 'grant_type=authorization_code' \
  -d 'client_id=dev-client' \
  -d 'client_secret=any-secret' \
  -d 'code=PASTE_CODE' \
  -d 'redirect_uri=http://127.0.0.1:9999/callback'
```

Response:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

Codes are **single-use** and expire after two minutes.

## 3. Userinfo

```bash
curl -s 'http://127.0.0.1:4190/google/userinfo' \
  -H 'Authorization: Bearer PASTE_ACCESS_TOKEN'
```

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

Repeat with `/acme/...` to get `"id": "acme:alice"`.
