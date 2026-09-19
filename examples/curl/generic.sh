#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

provider="${PROVIDER:-acme}"
code="$(authorize_code "$provider")"
token_response="$(exchange_code "$provider" "$code")"
access_token="$(printf '%s' "$token_response" | json_field access_token)"
userinfo="$(fetch_userinfo "$provider" "$access_token")"

printf '%s\n' "$userinfo" | python3 -c '
import json,sys
data=json.load(sys.stdin)
expected=sys.argv[1] + ":" + sys.argv[2]
assert data["id"] == expected, data
assert data["sub"] == expected, data
assert data["email"], data
print(json.dumps(data, indent=2))
' "$provider" "$PERSONA"
