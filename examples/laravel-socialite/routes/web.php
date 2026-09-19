<?php

use App\Http\Controllers\SocialLoginController;
use Illuminate\Support\Facades\Route;

Route::get('/', function () {
    return view('welcome');
});

Route::get('/login/{provider}', [SocialLoginController::class, 'redirect'])
    ->whereIn('provider', ['google', 'github']);

Route::get('/auth/{provider}/callback', [SocialLoginController::class, 'callback'])
    ->whereIn('provider', ['google', 'github']);
