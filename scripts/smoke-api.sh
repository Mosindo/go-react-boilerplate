#!/usr/bin/env bash
# End-to-end smoke test against a running API (real HTTP, real database):
# register x2 -> profile -> preferences -> photo -> discover -> like -> match -> chat -> read.
# Usage: scripts/smoke-api.sh [http://localhost:18080]
set -euo pipefail

API="${1:-http://localhost:18080}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
STAMP="$(date +%s)$RANDOM"

jget() { python3 -c 'import json,sys; d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    d = d[int(k)] if k.isdigit() else d[k]
print(d)' "$1"; }

call() { # method path token [json]
  local method="$1" path="$2" token="$3" body="${4:-}"
  local args=(-sS -X "$method" "$API$path" -H "Content-Type: application/json" -w '\n%{http_code}')
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$body" ] && args+=(-d "$body")
  curl "${args[@]}"
}
expect() { # expected_status response
  local code; code="$(echo "$2" | tail -n1)"
  if [ "$code" != "$1" ]; then echo "FAIL: expected HTTP $1, got $code: $(echo "$2" | sed '$d')" >&2; exit 1; fi
  echo "$2" | sed '$d'
}

python3 - "$TMP/photo.png" <<'PY'
import struct, sys, zlib
w, h = 300, 400
raw = b"".join(b"\x00" + b"".join(bytes((x % 256, y % 256, 128)) for x in range(w)) for y in range(h))
def chunk(t, d): return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xffffffff)
open(sys.argv[1], "wb").write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", w, h, 8, 2, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b""))
PY

echo "[1/9] health";   expect 200 "$(curl -sS -w '\n%{http_code}' "$API/health")" >/dev/null

onboard() { # name gender interestedIn -> prints "token id"
  local name="$1" gender="$2" into="$3" email="smoke-$1-$STAMP@smoke.test" out token id
  out="$(expect 201 "$(call POST /auth/register "" "{\"email\":\"$email\",\"password\":\"SmokeTest123\"}")")"
  token="$(echo "$out" | jget accessToken)"; id="$(echo "$out" | jget user.id)"
  expect 200 "$(call PUT /profile/me "$token" "{\"firstName\":\"$name\",\"birthDate\":\"1994-05-20\",\"gender\":\"$gender\",\"bio\":\"smoke\",\"interests\":[\"travel\",\"music\"]}")" >/dev/null
  expect 200 "$(call PUT /profile/preferences "$token" "{\"interestedIn\":[\"$into\"],\"minAge\":18,\"maxAge\":70,\"maxDistanceKm\":200}")" >/dev/null
  expect 201 "$(curl -sS -w '\n%{http_code}' -X POST "$API/photos" -H "Authorization: Bearer $token" -F "photo=@$TMP/photo.png;type=image/png")" >/dev/null
  echo "$token $id"
}

echo "[2/9] onboarding of two users"
read -r TOKEN_A ID_A <<<"$(onboard Alice woman man)"
read -r TOKEN_B ID_B <<<"$(onboard Bob man woman)"

echo "[3/9] discover"
expect 200 "$(call GET "/discover?limit=20" "$TOKEN_A")" | grep -q "$ID_B" || { echo "FAIL: Bob not in Alice's deck" >&2; exit 1; }

echo "[4/9] like (no match yet)"
R="$(expect 200 "$(call POST /swipes "$TOKEN_A" "{\"userId\":\"$ID_B\",\"action\":\"like\"}")")"
[ "$(echo "$R" | jget result)" = "liked" ] || { echo "FAIL: expected liked: $R" >&2; exit 1; }

echo "[5/9] reciprocal like -> match"
R="$(expect 200 "$(call POST /swipes "$TOKEN_B" "{\"userId\":\"$ID_A\",\"action\":\"like\"}")")"
[ "$(echo "$R" | jget result)" = "matched" ] || { echo "FAIL: expected matched: $R" >&2; exit 1; }
CONV="$(echo "$R" | jget match.conversationId)"

echo "[6/9] chat send"
expect 201 "$(call POST "/conversations/$CONV/messages" "$TOKEN_A" '{"body":"Bonjour !"}')" >/dev/null

echo "[7/9] chat list shows 1 unread for Bob"
R="$(expect 200 "$(call GET /conversations "$TOKEN_B")")"
[ "$(echo "$R" | jget conversations.0.unread)" = "1" ] || { echo "FAIL: expected unread=1: $R" >&2; exit 1; }

echo "[8/9] mark read + notifications"
expect 204 "$(call POST "/conversations/$CONV/read" "$TOKEN_B")" >/dev/null
expect 200 "$(call GET /notifications "$TOKEN_B")" >/dev/null

echo "[9/9] outsider cannot read the conversation"
read -r TOKEN_C _ <<<"$(onboard Eve woman man)"
expect 404 "$(call GET "/conversations/$CONV/messages" "$TOKEN_C")" >/dev/null

echo "SMOKE OK"
