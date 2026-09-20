# Provider profiles

Every slug returns the generic persona fields (`id`, `sub`, `email`,
`email_verified`, `name`, `nickname`, `avatar`). On top of that:

1. If `responseTemplate` is set, that template is used.
2. Otherwise, if the slug matches a built-in template (`github`, `google`, …),
   that template is used.
3. Otherwise the **default** template is used. It adds the aliases Socialite
   drivers commonly read (`login`, `username`, `preferred_username`,
   `display_name`, `picture`, `avatar_url`, `profile_image_url`), so an
   unlisted driver still works.

`responseTemplate: generic` skips the extra fields and returns only the
generic object.

Set `providerProfiles` in `bohurupee.yaml` to pick a template, add a custom
JSON object, or change endpoints and protocol. Same persona, different slug,
different body.

## Merge order

Later steps overwrite earlier keys (nested objects are merged):

1. Generic persona fields
2. Matched template (explicit `responseTemplate`, else the slug's built-in
   template, else `default`)
3. Custom profile `response:` map (if set)
4. Generic persona fields again (so `id` / `email` / `name` / `nickname` /
   `avatar` stay present for client getters)
5. Persona `claims:`
6. Persona `response:` overlay (can clear getters, including `avatar: ""`)

String values in profile or persona `response:` may use `{{nickname}}`,
`{{email}}`, `{{name}}`, `{{id}}`, `{{sub}}`, `{{avatar}}`, `{{provider}}`.

Persona extras in YAML:

```yaml
personas:
  - id: bob
    email: bob@example.com
    name: Bob User
    email_verified: false
    claims:
      role: user
    response:
      title: "{{nickname}}"
```

Omit `avatar` to keep the Dicebear default. Set `avatar: ""` for a missing
picture. `claims` also appear on the `id_token` (except reserved JWT keys).

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

`/acme/userinfo` uses the default template (`login`, `username`, …) and does
not include GitHub-only fields. `/github/userinfo` adds `login`, `avatar_url`,
`html_url` because the slug matches the `github` template, even with no
`providerProfiles` entry. `/staffdir/userinfo` is the default template plus
`title` and `username`.

## Built-in templates

| Name | Extra fields (typical) |
|------|------------------------|
| `default` | `login`, `username`, `preferred_username`, `display_name`, `picture`, `avatar_url`, `profile_image_url` |
| `generic` | no extra fields |
| `github` | `login`, `avatar_url`, `html_url`, `type` |
| `google` | `picture`, `given_name`, `family_name`, `verified_email` |
| `facebook` | `picture.data.url` |
| `twitch` | `login`, `display_name`, `profile_image_url` |
| `apple` | keeps generic userinfo; see protocol below |
| `jumpcloud` | `preferred_username`, `given_name`, `family_name`, `jc_org`, `member_of` |
| `gitlab` | `username`, `avatar_url`, `web_url`, `state` |
| `bitbucket` | `username`, `display_name`, `uuid`, `links.avatar.href` |
| `slack` | `user.name`, `user.real_name`, `user.profile.email`, `user.profile.image_192` |
| `linkedin` | `given_name`, `family_name`, `picture` |
| `discord` | `username`, `global_name`, `discriminator` |
| `microsoft` | `displayName`, `givenName`, `surname`, `mail`, `userPrincipalName` |

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

YAML field reference for `providerProfiles` (and every other key) lives in
[Configuration](configuration.md#providerprofiles).
