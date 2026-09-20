# Bohurupee (বহুরূপী)

<p class="hero-lede">Local fake identity provider for social login development. Point any OAuth or OIDC client at <code>http://127.0.0.1:4190</code> — nothing leaves your machine.</p>

**DEV ONLY.** Not a production IdP. See [Security](security.md).

## Quick path

```bash
brew install --cask milon/bohurupee/bohurupee
bohurupee init
bohurupee
```

Open [http://127.0.0.1:4190](http://127.0.0.1:4190/), pick a persona, finish the
code flow in your app.

## Documentation

| Guide                                     | Contents                                                |
|-------------------------------------------|---------------------------------------------------------|
| [Get started](get-started.md)             | Install (Homebrew, binary, Docker, source), first login |
| [Configuration](configuration.md)         | Every `bohurupee.yaml` key with examples                |
| [Endpoints and CLI](endpoints.md)         | Routes, query params, flags, env vars                   |
| [Any framework](any-framework.md)         | OIDC, Auth.js, CORS, reload, refresh                    |
| [Provider profiles](provider-profiles.md) | Templates, merge order, Apple / Facebook protocol       |
| [Laravel Socialite](socialite.md)         | Optional Packagist adapter                              |
| [Security](security.md)                   | Threat model and reporting                              |
| [Contributing](contributing.md)           | Develop, templates, docs, release                       |

## What it is

Bohurupee is one Go binary. Any URL-safe slug works (`/google`, `/github`,
`/acme`). Personas and provider shapes live in YAML. Laravel Socialite is
optional — any stack that speaks OAuth 2 / OIDC can use the same server.
