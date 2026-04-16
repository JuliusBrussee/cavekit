#!/usr/bin/env bash
# Thin compatibility shim around `cavekit command-gate`.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/cavekit" command-gate "$@"
