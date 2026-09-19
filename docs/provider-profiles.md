# Provider profiles

By default every slug (`/acme`, `/google`, …) returns the **generic** userinfo
object (`id`, `sub`, `email`, `email_verified`, `name`, `nickname`, `avatar`).

Set `providerProfiles` in `bohurupee.yaml` to overlay a built-in template and/or
a custom JSON object. Same persona, different slug, different body.

## Merge order

Later steps overwrite earlier keys (nested objects are merged):

1. Generic persona fields
2. Built-in `responseTemplate` (if set)
3. Custom `response:` map (if set)
4. Generic persona fields again (so `id` / `email` / `name` / `nickname` /
   `avatar` stay present for client getters)

String values in `response:` may use `{{nickname}}`, `{{email}}`, `{{name}}`,
`{{id}}`, `{{sub}}`, `{{avatar}}`, `{{provider}}`.

## Example

```yaml
providerProfiles:
  github:
    responseTemplate: github
  facebook:
    responseTemplate: facebook
    endpoints:
      userinfo: /me
  apple:
    responseTemplate: apple
    protocol:
      response_mode: form_post
      id_token: true
  staffdir:
    response:
      title: Engineer
      username: "{{nickname}}"
```

`/acme/userinfo` stays generic. `/github/userinfo` adds `login`, `avatar_url`,
`html_url`. `/staffdir/userinfo` is generic plus `title` and `username`.

## Built-in templates

| Name | Extra fields (typical) |
|------|------------------------|
| `github` | `login`, `avatar_url`, `html_url`, `type` |
| `google` | `picture`, `given_name`, `family_name`, `verified_email` |
| `facebook` | `picture.data.url` |
| `twitch` | `login`, `display_name`, `profile_image_url` |
| `apple` | keeps generic userinfo; see protocol below |

## `endpoints`

`endpoints.userinfo` registers an extra path on that provider. Facebook’s
`/me` becomes `GET /facebook/me` (same Bearer token as `/facebook/userinfo`).

## `protocol`

Used as **defaults** when the authorize query omits the matching parameter:

| Key | Effect |
|-----|--------|
| `response_mode: form_post` | Auto-POST `code`/`state` to `redirect_uri` (Apple-style) |
| `id_token: true` | Always include `id_token` on the token response, even without `openid` scope |

The client can still pass `response_mode=query` explicitly.
