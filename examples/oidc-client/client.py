#!/usr/bin/env python3
"""Dependency-free OIDC authorization-code client for Bohurupee."""

import base64
import hashlib
import json
import os
import secrets
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

BASE_URL = os.environ.get("BOHURUPEE_URL", "http://127.0.0.1:4190").rstrip("/")
PROVIDER = os.environ.get("PROVIDER", "google")
PERSONA = os.environ.get("PERSONA", "alice")
CLIENT_ID = os.environ.get("CLIENT_ID", "python-oidc-example")
REDIRECT_URI = os.environ.get("REDIRECT_URI", "http://127.0.0.1:9999/callback")


def get_json(url, headers=None):
    request = urllib.request.Request(url, headers=headers or {})
    with urllib.request.urlopen(request) as response:
        return json.load(response)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def b64url_decode(value):
    return base64.urlsafe_b64decode(value + "=" * (-len(value) % 4))


def verify_rs256(compact_jwt, jwks):
    header_segment, payload_segment, signature_segment = compact_jwt.split(".")
    header = json.loads(b64url_decode(header_segment))
    if header.get("alg") != "RS256":
        raise ValueError(f"expected RS256, got {header.get('alg')!r}")

    key = next((item for item in jwks["keys"] if item.get("kid") == header.get("kid")), None)
    if key is None:
        raise ValueError("signing key not found in JWKS")

    modulus = int.from_bytes(b64url_decode(key["n"]), "big")
    exponent = int.from_bytes(b64url_decode(key["e"]), "big")
    signature = int.from_bytes(b64url_decode(signature_segment), "big")
    encoded = pow(signature, exponent, modulus).to_bytes((modulus.bit_length() + 7) // 8, "big")

    signing_input = f"{header_segment}.{payload_segment}".encode()
    digest = hashlib.sha256(signing_input).digest()
    sha256_digest_info = bytes.fromhex("3031300d060960864801650304020105000420") + digest
    padding_length = len(encoded) - len(sha256_digest_info) - 3
    expected = b"\x00\x01" + b"\xff" * padding_length + b"\x00" + sha256_digest_info
    if encoded != expected:
        raise ValueError("id_token signature verification failed")

    return json.loads(b64url_decode(payload_segment))


def main():
    discovery_url = f"{BASE_URL}/{PROVIDER}/.well-known/openid-configuration"
    discovery = get_json(discovery_url)

    verifier = secrets.token_urlsafe(48)
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    state = secrets.token_urlsafe(18)
    nonce = secrets.token_urlsafe(18)
    query = urllib.parse.urlencode(
        {
            "client_id": CLIENT_ID,
            "redirect_uri": REDIRECT_URI,
            "response_type": "code",
            "scope": "openid profile email",
            "state": state,
            "nonce": nonce,
            "code_challenge": challenge,
            "code_challenge_method": "S256",
            "auto": PERSONA,
        }
    )

    opener = urllib.request.build_opener(NoRedirect)
    try:
        opener.open(f"{discovery['authorization_endpoint']}?{query}")
        raise RuntimeError("authorize did not redirect")
    except urllib.error.HTTPError as error:
        if error.code not in (302, 303):
            raise
        location = error.headers["Location"]

    callback = urllib.parse.urlparse(location)
    callback_query = urllib.parse.parse_qs(callback.query)
    if callback_query.get("state", [None])[0] != state:
        raise ValueError("state mismatch")
    code = callback_query["code"][0]

    token_body = urllib.parse.urlencode(
        {
            "grant_type": "authorization_code",
            "client_id": CLIENT_ID,
            "client_secret": "any-secret",
            "redirect_uri": REDIRECT_URI,
            "code": code,
            "code_verifier": verifier,
        }
    ).encode()
    request = urllib.request.Request(discovery["token_endpoint"], data=token_body, method="POST")
    with urllib.request.urlopen(request) as response:
        tokens = json.load(response)

    jwks = get_json(discovery["jwks_uri"])
    claims = verify_rs256(tokens["id_token"], jwks)
    audience = claims.get("aud", [])
    if isinstance(audience, str):
        audience = [audience]
    assert claims["iss"] == discovery["issuer"], claims
    assert CLIENT_ID in audience, claims
    assert claims["nonce"] == nonce, claims
    assert claims["exp"] > time.time(), claims

    userinfo = get_json(
        discovery["userinfo_endpoint"],
        {"Authorization": f"Bearer {tokens['access_token']}"},
    )
    assert claims["sub"] == userinfo["sub"], (claims, userinfo)

    print("Discovery:", discovery_url)
    print("Verified id_token subject:", claims["sub"])
    print("Userinfo:")
    print(json.dumps(userinfo, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        print(f"OIDC example failed: {error}", file=sys.stderr)
        raise
