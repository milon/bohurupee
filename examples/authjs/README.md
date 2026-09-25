# Auth.js + Bohurupee

Copy [`provider.ts`](provider.ts) into your app (or import it from this folder
in a monorepo). It is a normal Auth.js OIDC provider: discovery + PKCE, no
Bohurupee SDK.

```bash
# terminal 1
bohurupee --config ./bohurupee.example.yaml

# terminal 2 — Next.js / Auth.js on :3000
```

```ts
import NextAuth from "next-auth"
import Google from "next-auth/providers/google"
import { Bohurupee } from "./bohurupee" // copy of provider.ts

export const { handlers, auth } = NextAuth({
  providers: [
    ...(process.env.NODE_ENV === "development"
      ? [Bohurupee({ provider: "google" })]
      : [Google({ clientId: process.env.GOOGLE_ID!, clientSecret: process.env.GOOGLE_SECRET! })]),
  ],
})
```

Register the callback your Auth.js app already uses, for example
`http://localhost:3000/api/auth/callback/bohurupee`. Bohurupee accepts any
local `client_id` / secret.

Loopback CORS is enabled on discovery, token, and userinfo so a browser app on
another localhost port can finish the code flow.

See [`docs/content/any-framework.md`](../../docs/content/any-framework.md) for the same
provider inline.
