# Contributing

Bohurupee is a local fake identity provider. Keep the default userinfo
generic. Provider-shaped JSON is opt-in.

```bash
go test ./...
```

## Add a response template

Built-in shapes live in `internal/profiles/templates.go`. A template adds
fields on top of the generic persona; it does not replace `id`, `email`,
`name`, `nickname`, or `avatar`.

1. Add a function with the same signature as `githubTemplate`.
2. Register it in the `templates` map. The key is the YAML name (`github`,
   `google`, …) and must be lowercase.
3. Add a case in `internal/profiles/render_test.go`. Assert the new fields
   and that the generic getters are still present.
4. Add a row to the table in `docs/provider-profiles.md`.
5. Optionally add a `providerProfiles` entry in `bohurupee.example.yaml`.

Unknown `responseTemplate` values are rejected at startup.

You do not need a code change for a one-off shape. Set `response:` in
`bohurupee.yaml` instead. See `docs/provider-profiles.md`.
