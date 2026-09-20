# Security Policy

Bohurupee is a **DEV ONLY** fake identity provider. It is not a production
IdP. Codes and tokens live in process memory. Any `client_id` and
`client_secret` are accepted by default. Binding beyond loopback is a
deliberate footgun (`--dangerously-bind-all-interfaces` or Docker publish
mistakes).

That does not mean security reports are unwelcome.

## Supported versions

Only the latest release line receives fixes. Older tags are historical.

## Reporting a vulnerability

Please **do not** open a public GitHub issue for anything that could put
users at risk if misused (for example a way to escape the loopback guard,
tamper with tokens across origins, or confuse a real OAuth client into
trusting Bohurupee in production).

Report privately via
[GitHub Security Advisories](https://github.com/milon/bohurupee/security/advisories/new)
for this repository.

Include Bohurupee version (`bohurupee --version`), OS, and steps to
reproduce. We will acknowledge and decide whether a fix, docs change, or
wontfix (out of threat model) is appropriate.

## Out of scope

- Exposing Bohurupee on a LAN or the public internet
- Using Bohurupee as a production identity provider
- Validating real third-party client secrets (Apple, etc.)
