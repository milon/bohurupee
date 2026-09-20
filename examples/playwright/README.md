# Playwright `loginAs`

Call this **before** the app starts the OAuth redirect. It does not open the
consent page. The helper posts to Bohurupee's `/__login` route using the
Playwright browser context so the auto-approve cookie is sent on the next
`/{provider}/authorize` request.

```js
import { test } from "@playwright/test";
import { loginAs } from "./loginAs.mjs";

test("signs in as Alice", async ({ page, context }) => {
  await loginAs(context, "alice", "google");
  await page.goto("http://127.0.0.1:8000/auth/google");
  // app callback receives google:alice
});
```

`persona` must exist in `bohurupee.yaml`. `provider` is recorded in the JSON
response; the cookie applies to every slug.

To test cancel, do not call `loginAs`. Open authorize and click **Deny**, or
add `deny=1` to the authorize URL. The app should see `error=access_denied`.
