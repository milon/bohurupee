#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Generic provider:"
"${SCRIPT_DIR}/generic.sh"

echo
echo "GitHub-shaped provider:"
"${SCRIPT_DIR}/github.sh"
