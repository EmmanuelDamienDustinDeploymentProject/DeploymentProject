#!/bin/bash

# MCP OAuth / tool exercise script
# Works best with stateless mode (default). Set MCP_STATELESS=0 to force sessionful mode.
# Requires TOKEN (export TOKEN=...). Use /oauth/login in a browser to obtain one.

set -euo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
TOKEN=${TOKEN:-}
VERBOSE=${VERBOSE:-}

bold() { printf "\e[1m%s\e[0m\n" "$*"; }
section() { echo; echo "--------------------------------------------------"; bold "$1"; echo "--------------------------------------------------"; }
info() { printf "[info] %s\n" "$*"; }
warn() { printf "[warn] %s\n" "$*"; }
err() { printf "[error] %s\n" "$*" >&2; }

section "Environment"
info "BASE_URL=$BASE_URL"
if [ -n "${MCP_STATELESS:-}" ]; then info "MCP_STATELESS=$MCP_STATELESS"; else info "MCP_STATELESS (unset => stateless default)"; fi
if [ -n "$TOKEN" ]; then info "TOKEN provided"; else warn "TOKEN missing: visit $BASE_URL/oauth/login to authenticate, then export TOKEN=<bearer_token>"; fi

section "1. Health check (no auth)"
curl -s -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/health" || err "Health check failed"

section "2. Unauth /mcp (expect 401)"
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" "$BASE_URL/mcp"

section "3. Invalid token /mcp (expect 401)"
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" -H "Authorization: Bearer invalid_token_here" "$BASE_URL/mcp"

if [ -z "$TOKEN" ]; then
  warn "Skipping authenticated calls; TOKEN not set."
  exit 0
fi

AUTH=(-H "Authorization: Bearer $TOKEN")

section "4. REST /time (nyc default)"
curl -s "${AUTH[@]}" -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/time" || err "/time default failed"
section "5. REST /time city=sf"
curl -s "${AUTH[@]}" -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/time?city=sf" || err "/time sf failed"
section "6. REST /time city=boston"
curl -s "${AUTH[@]}" -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/time?city=boston" || err "/time boston failed"

section "7. REST /whoami"
curl -s "${AUTH[@]}" -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/whoami" || err "/whoami failed"

# JSON-RPC tool calls.
section "8. tools/list"
LIST_RESP=$(curl -s "${AUTH[@]}" -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":"list-1","method":"tools/list","params":{}}' "$BASE_URL/mcp")
printf "%s\n" "$LIST_RESP" | jq . 2>/dev/null || printf "%s\n" "$LIST_RESP"
# Detect sessionful error pattern
if echo "$LIST_RESP" | grep -qi 'invalid during session initialization'; then
  warn "Server reports session initialization required. Enable stateless mode: export MCP_STATELESS=1 and restart."
fi

section "9. tools/call cityTime (sf)"
CALL_SF=$(curl -s "${AUTH[@]}" -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"cityTime","arguments":{"city":"sf"}}}' "$BASE_URL/mcp")
printf "%s\n" "$CALL_SF" | jq . 2>/dev/null || printf "%s\n" "$CALL_SF"

section "10. tools/call cityTime (unknown -> expect error)"
CALL_BAD=$(curl -s "${AUTH[@]}" -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":"call-2","method":"tools/call","params":{"name":"cityTime","arguments":{"city":"unknown"}}}' "$BASE_URL/mcp")
printf "%s\n" "$CALL_BAD" | jq . 2>/dev/null || printf "%s\n" "$CALL_BAD"

section "Summary"
info "If tools/list showed an initialization error, restart server with: export MCP_STATELESS=1; go run ."
info "Done."
