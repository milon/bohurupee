/**
 * Drop-in Auth.js / NextAuth OIDC provider for local Bohurupee.
 *
 * Usage (Auth.js v5 / NextAuth):
 *
 *   import NextAuth from "next-auth"
 *   import { Bohurupee } from "./bohurupee" // or copy from this file
 *
 *   export const { handlers, auth } = NextAuth({
 *     providers: [
 *       ...(process.env.NODE_ENV === "development"
 *         ? [Bohurupee({ provider: "google" })]
 *         : [Google({ clientId: "...", clientSecret: "..." })]),
 *     ],
 *   })
 *
 * Start Bohurupee first: `bohurupee --config ./bohurupee.example.yaml`
 * Callback URL: http://localhost:3000/api/auth/callback/bohurupee
 */

export type BohurupeeOptions = {
  /** Provider slug on Bohurupee (google, github, acme, …). Default: "google". */
  provider?: string
  /** Bohurupee origin. Default: http://127.0.0.1:4190 */
  origin?: string
  clientId?: string
  clientSecret?: string
}

export function Bohurupee(options: BohurupeeOptions = {}) {
  const provider = options.provider ?? "google"
  const origin = (options.origin ?? "http://127.0.0.1:4190").replace(/\/$/, "")
  const issuer = `${origin}/${provider}`

  return {
    id: "bohurupee",
    name: "Bohurupee",
    type: "oidc" as const,
    issuer,
    wellKnown: `${issuer}/.well-known/openid-configuration`,
    clientId: options.clientId ?? "authjs-local",
    clientSecret: options.clientSecret ?? "dev-secret",
    authorization: { params: { scope: "openid profile email" } },
    checks: ["pkce", "state"] as ("pkce" | "state")[],
    profile(profile: {
      sub?: string
      name?: string
      email?: string
      picture?: string
      avatar?: string
    }) {
      return {
        id: profile.sub ?? "",
        name: profile.name ?? null,
        email: profile.email ?? null,
        image: profile.picture ?? profile.avatar ?? null,
      }
    },
  }
}
