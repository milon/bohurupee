# Use Bohurupee from any framework

Bohurupee is a local OAuth 2.0 / OpenID Connect server. Your application only
needs to support a custom OAuth or OIDC provider; it does not need Go, PHP, or a
Bohurupee-specific SDK.

## Choose a provider slug

Any URL-safe slug works:

```text
http://127.0.0.1:4190/acme
http://127.0.0.1:4190/google
http://127.0.0.1:4190/github
```

The slug namespaces identities (`google:alice`, `github:alice`). Unless the
slug has a configured provider profile, userinfo uses the generic shape.

## OIDC (recommended)

Point the client at this issuer:

```text
http://127.0.0.1:4190/{provider}
```

Its discovery document is:

```text
http://127.0.0.1:4190/{provider}/.well-known/openid-configuration
```

Use authorization code flow, request `openid profile email`, and enable PKCE
S256. Bohurupee accepts any local `client_id` and any secret.

The dependency-free Python example demonstrates discovery, PKCE, token
exchange, JWKS signature verification, and userinfo:

```bash
python3 examples/oidc-client/client.py
```

## OAuth-only clients

Clients without OIDC discovery can configure the endpoints directly:

- Authorization: `GET /{provider}/authorize`
- Token: `POST /{provider}/token`
- Userinfo: `GET /{provider}/userinfo`
- JWKS: `GET /{provider}/jwks`

Required authorization parameters are `client_id`, `redirect_uri`,
`response_type=code`, and `state`. `scope=openid` adds an `id_token`.

The client may authenticate at the token endpoint with HTTP Basic or
`client_id` / `client_secret` form fields. Bohurupee is an open local client, so
the secret is accepted but not validated.

## Auth.js

Configure a custom OIDC provider. This illustrates the relevant Auth.js
settings; place it inside the providers array used by your Auth.js version:

```ts
{
  id: "bohurupee",
  name: "Bohurupee",
  type: "oidc",
  issuer: "http://127.0.0.1:4190/acme",
  wellKnown:
    "http://127.0.0.1:4190/acme/.well-known/openid-configuration",
  clientId: "authjs-local",
  clientSecret: "dev-secret",
  authorization: { params: { scope: "openid profile email" } },
  checks: ["pkce", "state"],
  profile(profile) {
    return {
      id: profile.sub,
      name: profile.name,
      email: profile.email,
      image: profile.picture ?? profile.avatar,
    }
  },
}
```

Set these URLs only in local development. Keep the real provider configuration
for production.

## Generic framework checklist

1. Start Bohurupee on loopback.
2. Set the framework's issuer or endpoint URLs to Bohurupee.
3. Use any client ID and secret.
4. Register your application's normal local callback URL as `redirect_uri`.
5. Enable authorization code flow, state, and PKCE S256.
6. Request `openid profile email` for OIDC.
7. Map `sub` or `id` as the user ID; map `email`, `name`, `nickname`, and
   `picture` / `avatar` as needed.
8. Never enable the local provider in production.

## Automated login

The browser consent page lets a developer select a configured persona, or
click **Deny** (`error=access_denied` on `redirect_uri`). Tests can:

- add `auto=alice` to the authorization request
- set `BOHURUPEE_AUTO_APPROVE=1` for the default persona
- `POST /__login` with `persona=alice` so the next authorize in that cookie
  jar skips consent without rewriting the authorize URL

Playwright: [`examples/playwright`](../examples/playwright). PHP:
[`examples/php`](../examples/php).

See also:

- [curl examples](../examples/curl/README.md)
- [provider profiles](provider-profiles.md)
- [Laravel Socialite adapter](socialite.md)
