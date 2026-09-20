# PHP `loginAs`

`loginAs.php` posts to `POST /__login` and returns the `Set-Cookie` value.
Attach that cookie when your test HTTP client follows authorize on the
Bohurupee origin. Laravel Socialite browser tests should use the Playwright
helper instead, because the cookie belongs on `127.0.0.1:4190`, not on the
Laravel app.

```php
require __DIR__.'/loginAs.php';

$session = bohurupee_login_as('alice', 'google');
$opts = [
    'http' => [
        'header' => 'Cookie: '.$session['cookie']."\r\n",
        'follow_location' => 0,
    ],
];
$authorize = 'http://127.0.0.1:4190/google/authorize?'.http_build_query([
    'client_id' => 'dev-client',
    'redirect_uri' => 'http://127.0.0.1:9999/callback',
    'response_type' => 'code',
    'state' => 'xyz',
]);
file_get_contents($authorize, false, stream_context_create($opts));
```
