---
title: Welcome
pretoc: true
---

# Bohurupee (বহুরূপী)

<img src="assets/logo.svg" alt="" width="88">

Local fake identity provider for social login development. Point any OAuth or
OIDC client at `http://127.0.0.1:4190` — nothing leaves your machine.

**DEV ONLY.** Not a production IdP. See [Security](security.html).

## Quick path

```bash
brew install --cask milon/bohurupee/bohurupee
bohurupee init
bohurupee
```

Open [http://127.0.0.1:4190](http://127.0.0.1:4190/), pick a persona, finish the
code flow in your app.

## What it is

Bohurupee is one Go binary. Any URL-safe slug works (`/google`, `/github`,
`/acme`). Personas and provider shapes live in YAML. Laravel Socialite is
optional — any stack that speaks OAuth 2 / OIDC can use the same server.

Continue with [Get started](get-started.html).
