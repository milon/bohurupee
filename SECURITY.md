# Security Policy

Bohurupee is a **DEV ONLY** fake identity provider for local development. It
is not a production IdP. Codes and tokens live in process memory. Any
`client_id` and `client_secret` are accepted by default. Binding beyond
loopback is a deliberate footgun (`--dangerously-bind-all-interfaces` or
Docker publish mistakes).

## Supported versions

Only the latest release line receives fixes. Older tags are historical.

## Reporting

Open a [GitHub issue](https://github.com/milon/bohurupee/issues) for bugs and
docs problems. Include `bohurupee --version`, OS, and steps to reproduce when
you can.

## Out of scope

- Exposing Bohurupee on a LAN or the public internet
- Using Bohurupee as a production identity provider
- Validating real third-party client secrets (Apple, etc.)
