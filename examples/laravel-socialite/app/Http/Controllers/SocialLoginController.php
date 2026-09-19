<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Laravel\Socialite\Facades\Socialite;
use Symfony\Component\HttpFoundation\RedirectResponse as SymfonyRedirect;

class SocialLoginController
{
    public function redirect(string $provider): RedirectResponse|SymfonyRedirect
    {
        return Socialite::driver($provider)->redirect();
    }

    public function callback(string $provider): JsonResponse
    {
        $user = Socialite::driver($provider)->user();

        return response()->json([
            'provider' => $provider,
            'id' => $user->getId(),
            'email' => $user->getEmail(),
            'name' => $user->getName(),
            'nickname' => $user->getNickname(),
            'avatar' => $user->getAvatar(),
            'raw' => $user->getRaw(),
        ]);
    }
}
