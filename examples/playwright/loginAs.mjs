/**
 * Skip the Bohurupee consent page in Playwright.
 *
 * POST /__login sets an HttpOnly cookie on the Bohurupee origin. The next
 * authorize request from this browser context continues as that persona.
 *
 * @param {import('@playwright/test').BrowserContext | import('@playwright/test').Page} ctx
 * @param {string} persona
 * @param {string} [provider]
 * @param {string} [origin]
 */
export async function loginAs(ctx, persona, provider = "google", origin = "http://127.0.0.1:4190") {
  const request = "request" in ctx && ctx.request ? ctx.request : ctx.context().request;
  const res = await request.post(`${origin.replace(/\/$/, "")}/__login`, {
    form: { persona, provider },
  });
  if (!res.ok()) {
    throw new Error(`bohurupee loginAs ${persona}: ${res.status()} ${await res.text()}`);
  }
  return res.json();
}
