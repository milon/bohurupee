# Laravel Socialite example

Slim Laravel app that uses `milon/bohurupee-laravel` so Google and GitHub
login hit a local Bohurupee process.

```bash
# from the repo root
go build -o ./bohurupee ./cmd/bohurupee
./bohurupee --config ./bohurupee.example.yaml

cd examples/laravel-socialite
composer install
cp .env.example .env
php artisan key:generate
php artisan serve
```

Open [http://127.0.0.1:8000](http://127.0.0.1:8000). After consent, the callback
prints the Socialite user as JSON (`id`, `email`, `name`, `nickname`, `avatar`,
and `raw`). Deny (and other OAuth failures) are handled by
`milon/bohurupee-laravel` as `OAuthErrorException` and returned as JSON
`error` / `error_description` instead of Laravel’s exception page.

Requires PHP 8.2+ and Composer. See [`docs/socialite.md`](../../docs/socialite.md).
