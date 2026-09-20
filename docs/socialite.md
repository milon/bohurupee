# Laravel Socialite adapter

Point Socialite at a local Bohurupee process so Google, GitHub, and custom
slugs never leave your machine.

The adapter is a separate Composer package, `milon/bohurupee-laravel`
(local checkout: `../bohurupee-laravel`). It is optional: any other stack can
use the same OAuth/OIDC server with [`docs/any-framework.md`](any-framework.md).

## Install

```bash
composer require milon/bohurupee-laravel --dev
```

From a sibling checkout, path-require it:

```json
{
  "require-dev": {
    "milon/bohurupee-laravel": "@dev"
  },
  "repositories": [
    {
      "type": "path",
      "url": "../bohurupee-laravel"
    }
  ]
}
```

Laravel auto-discovers `Milon\Bohurupee\BohurupeeServiceProvider`.

## Environment

| Variable | Default | Meaning |
|----------|---------|---------|
| `BOHURUPEE_ENABLED` | `false` | Wrap Socialite when `true` |
| `BOHURUPEE_URL` | `http://127.0.0.1:4190` | Bohurupee origin (no trailing slash needed) |
| `BOHURUPEE_DRIVERS` | empty | Comma-separated Socialite names to wrap. Empty means every `Socialite::driver($name)` |
| `BOHURUPEE_EXCEPT` | empty | Names that keep the native Socialite provider |

Publish the config if you prefer PHP over env:

```bash
php artisan vendor:publish --tag=bohurupee-config
```

Keep `config/services.php` as usual (`client_id`, `client_secret`, `redirect`).
Bohurupee is an open local client: any secret is accepted.

## Production guard

If `BOHURUPEE_ENABLED=true` and `APP_ENV=production`, the provider throws
`Milon\Bohurupee\ProductionForbiddenException` at boot. Leave the flag off in
deployed environments, or require the package as `require-dev` only.

## How wrapping works

When enabled, Socialite's factory is decorated. `Socialite::driver('google')`
(and `github`, `facebook`, `acme`, …) returns `BohurupeeProvider` unless the
name is excluded. That provider uses:

- `GET {BOHURUPEE_URL}/{name}/authorize`
- `POST {BOHURUPEE_URL}/{name}/token`
- `GET {BOHURUPEE_URL}/{name}/userinfo`

Start Bohurupee first:

```bash
./bohurupee --config ./bohurupee.example.yaml
```

Then start your app and hit the same login routes you already use.

## User mapping

Userinfo JSON is applied with `setRaw()`. Mapped fields, first match wins:

| Socialite | userinfo keys |
|-----------|----------------|
| `id` | `id`, `sub` |
| `email` | `email` |
| `name` | `name`, `display_name` |
| `nickname` | `nickname`, `login`, `preferred_username` |
| `avatar` | `picture.data.url`, `picture`, `avatar`, `profile_image_url`, `avatar_url` |

GitHub profiles therefore expose `login` / `avatar_url` on `$user->getRaw()`
and on the mapped nickname/avatar.

## Example app

[`examples/laravel-socialite`](../examples/laravel-socialite) is a slim Laravel
app with Google and GitHub login buttons.

```bash
./bohurupee --config ./bohurupee.example.yaml
cd examples/laravel-socialite
composer install
cp .env.example .env
php artisan key:generate
php artisan serve
```

Open `http://127.0.0.1:8000` and choose a provider. Consent on Bohurupee
returns JSON for the Socialite user.

## Tests

Package tests mock Guzzle so token and userinfo never touch `google.com` or
`github.com`:

```bash
cd ../bohurupee-laravel
composer install
vendor/bin/phpunit
```

Browser tests can skip consent with
[`examples/playwright`](../examples/playwright) (`loginAs(context, 'alice', 'google')`)
or [`examples/php`](../examples/php).
