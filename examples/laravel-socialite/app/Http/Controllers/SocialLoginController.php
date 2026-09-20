<?php

namespace App\Http\Controllers;

use Illuminate\Http\JsonResponse;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;
use Laravel\Socialite\Facades\Socialite;
use Symfony\Component\HttpFoundation\RedirectResponse as SymfonyRedirect;
use Throwable;

class SocialLoginController
{
    public function redirect(string $provider): RedirectResponse|SymfonyRedirect
    {
        return Socialite::driver($provider)->redirect();
    }

    public function callback(Request $request, string $provider): JsonResponse
    {
        if ($request->filled('error')) {
            return response()->json([
                'provider' => $provider,
                'error' => $request->query('error'),
                'error_description' => $request->query('error_description'),
                'state' => $request->query('state'),
            ], 400);
        }

        try {
            $user = Socialite::driver($provider)->user();
        } catch (Throwable $e) {
            return response()->json([
                'provider' => $provider,
                'error' => class_basename($e),
                'error_description' => $e->getMessage() !== '' ? $e->getMessage() : class_basename($e),
            ], 400);
        }

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
