#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

provider="github"
code="$(authorize_code "$provider")"
token_response="$(exchange_code "$provider" "$code")"
access_token="$(printf '%s' "$token_response" | json_field access_token)"
userinfo="$(fetch_userinfo "$provider" "$access_token")"

printf '%s\n' "$userinfo" | python3 -c '
import json,sys
data=json.load(sys.stdin)
expected="github:" + sys.argv[1]
assert data["id"] == expected, data
assert data["login"] == sys.argv[1], data
assert data["avatar_url"], data
print(json.dumps(data, indent=2))
' "$PERSONA"
