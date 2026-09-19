<?php

return [
    'google' => [
        'client_id' => env('GOOGLE_CLIENT_ID', 'bohurupee-google'),
        'client_secret' => env('GOOGLE_CLIENT_SECRET', 'bohurupee'),
        'redirect' => env('GOOGLE_REDIRECT_URI', 'http://127.0.0.1:8000/auth/google/callback'),
    ],
    'github' => [
        'client_id' => env('GITHUB_CLIENT_ID', 'bohurupee-github'),
        'client_secret' => env('GITHUB_CLIENT_SECRET', 'bohurupee'),
        'redirect' => env('GITHUB_REDIRECT_URI', 'http://127.0.0.1:8000/auth/github/callback'),
    ],
];
