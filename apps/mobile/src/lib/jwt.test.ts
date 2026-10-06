import assert from "node:assert/strict";
import { test } from "node:test";
import { isExpiringSoon, jwtExpiry } from "./jwt.ts";

function token(payload: object): string {
  const b64 = (value: object) => Buffer.from(JSON.stringify(value)).toString("base64url");
  return `${b64({ alg: "HS256", typ: "JWT" })}.${b64(payload)}.signature`;
}

test("jwtExpiry reads exp without verifying", () => {
  assert.equal(jwtExpiry(token({ exp: 1_900_000_000 })), 1_900_000_000);
  assert.equal(jwtExpiry(token({ uid: "x" })), null);
  assert.equal(jwtExpiry("garbage"), null);
  assert.equal(jwtExpiry("a.%%%.c"), null);
});

test("isExpiringSoon applies a safety skew", () => {
  const now = Date.UTC(2026, 0, 1);
  const inTenMinutes = Math.floor(now / 1000) + 600;
  const inTenSeconds = Math.floor(now / 1000) + 10;
  assert.equal(isExpiringSoon(token({ exp: inTenMinutes }), now), false);
  assert.equal(isExpiringSoon(token({ exp: inTenSeconds }), now), true);
  assert.equal(isExpiringSoon("not-a-jwt", now), true);
});
