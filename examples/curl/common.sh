#!/usr/bin/env bash
set -euo pipefail

BOHURUPEE_URL="${BOHURUPEE_URL:-http://127.0.0.1:4190}"
CLIENT_ID="${CLIENT_ID:-example-client}"
REDIRECT_URI="${REDIRECT_URI:-http://127.0.0.1:9999/callback}"
PERSONA="${PERSONA:-alice}"

json_field() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)[sys.argv[1]])' "$1"
}

authorize_code() {
  local provider="$1"
  local headers location
  headers="$(curl --fail --silent --show-error --dump-header - --output /dev/null \
    --get "${BOHURUPEE_URL}/${provider}/authorize" \
    --data-urlencode "client_id=${CLIENT_ID}" \
    --data-urlencode "redirect_uri=${REDIRECT_URI}" \
    --data-urlencode "response_type=code" \
    --data-urlencode "state=example-state" \
    --data-urlencode "auto=${PERSONA}")"
  location="$(printf '%s\n' "$headers" | awk 'tolower($1) == "location:" {sub(/\r$/, "", $2); print $2}')"
  python3 -c 'import sys,urllib.parse; print(urllib.parse.parse_qs(urllib.parse.urlparse(sys.argv[1]).query)["code"][0])' "$location"
}

exchange_code() {
  local provider="$1"
  local code="$2"
  curl --fail --silent --show-error \
    --request POST "${BOHURUPEE_URL}/${provider}/token" \
    --user "${CLIENT_ID}:any-secret" \
    --data-urlencode "grant_type=authorization_code" \
    --data-urlencode "code=${code}" \
    --data-urlencode "redirect_uri=${REDIRECT_URI}"
}

fetch_userinfo() {
  local provider="$1"
  local token="$2"
  curl --fail --silent --show-error \
    "${BOHURUPEE_URL}/${provider}/userinfo" \
    --header "Authorization: Bearer ${token}"
}
