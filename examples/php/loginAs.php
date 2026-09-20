<?php

/**
 * Skip the Bohurupee consent page from PHP tests.
 *
 * Posts to /__login and returns the Set-Cookie header value so you can attach
 * it to the next HTTP client request to /{provider}/authorize. Playwright
 * and real browsers should use examples/playwright/loginAs.mjs instead: the
 * cookie must be stored for the Bohurupee origin, not the app origin.
 */
function bohurupee_login_as(
    string $persona,
    string $provider = 'google',
    string $origin = 'http://127.0.0.1:4190',
): array {
    $origin = rtrim($origin, '/');
    $body = http_build_query(['persona' => $persona, 'provider' => $provider]);
    $ctx = stream_context_create([
        'http' => [
            'method' => 'POST',
            'header' => "Content-Type: application/x-www-form-urlencoded\r\n",
            'content' => $body,
            'ignore_errors' => true,
        ],
    ]);
    $raw = file_get_contents($origin.'/__login', false, $ctx);
    if ($raw === false) {
        throw new RuntimeException('bohurupee loginAs: request failed');
    }
    $status = 0;
    $cookie = '';
    foreach ($http_response_header ?? [] as $line) {
        if (preg_match('/^HTTP\/\S+\s+(\d+)/', $line, $m)) {
            $status = (int) $m[1];
        }
        if (stripos($line, 'Set-Cookie:') === 0) {
            $cookie = trim(substr($line, strlen('Set-Cookie:')));
        }
    }
    if ($status < 200 || $status >= 300) {
        throw new RuntimeException("bohurupee loginAs {$persona}: HTTP {$status} {$raw}");
    }
    $json = json_decode($raw, true, 512, JSON_THROW_ON_ERROR);

    return [
        'persona' => $json['persona'] ?? $persona,
        'provider' => $json['provider'] ?? $provider,
        'cookie' => $cookie,
    ];
}
