# Security

Bohurupee is a **DEV ONLY** fake identity provider. It is meant to run on
your machine, on loopback, while you develop OAuth / OIDC clients. It is not
a production IdP and does not try to be one.

## What that means

- Codes and tokens live in **process memory** and disappear when the process exits
- Any `client_id` / `client_secret` are accepted by default (`openClient`)
- The default bind is loopback (`127.0.0.1`). Non-loopback needs an explicit
  override (`--dangerously-bind-all-interfaces` or a bad Docker publish)
- Do not point a production application at it

Docker: publish the host port on `127.0.0.1` only
(`-p 127.0.0.1:4190:4190`).

Config knobs such as `openClient`, `clients`, and `pkce` are documented in
[Configuration](configuration.md).

## Bugs and reports

File a normal [GitHub issue](https://github.com/milon/bohurupee/issues) for
bugs, confusing behavior, or docs gaps. This project only works locally by
design.

Repo policy:
[SECURITY.md](https://github.com/milon/bohurupee/blob/master/SECURITY.md).
