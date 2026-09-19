# Python OIDC client

This dependency-free Python example discovers Bohurupee's endpoints, performs
authorization code + PKCE, exchanges the code, verifies the RS256 `id_token`
against JWKS, and fetches userinfo.

Start Bohurupee from the repository root:

```bash
go run ./cmd/bohurupee --config ./bohurupee.example.yaml
```

In another terminal:

```bash
python3 examples/oidc-client/client.py
```

Optional environment variables:

```bash
BOHURUPEE_URL=http://127.0.0.1:4190 \
PROVIDER=acme \
PERSONA=bob \
python3 examples/oidc-client/client.py
```

`auto=<persona>` makes this example non-interactive. A real application should
send the browser to the discovered authorization endpoint so the developer can
pick a persona on the consent page.
