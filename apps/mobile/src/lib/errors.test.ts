import { describe, expect, it } from "vitest";
import { errorStatus, isStatus, messageFromError, parseRetryAfter } from "./errors";

function apiError(status: number, message: string, extra: Record<string, unknown> = {}) {
  return Object.assign(new Error(message), { status, ...extra });
}

describe("messageFromError", () => {
  it("uses the server message for 4xx", () => {
    expect(messageFromError(apiError(409, "email already registered"))).toBe("email already registered");
    expect(messageFromError(apiError(422, "you must be 18"))).toBe("you must be 18");
  });
  it("hides 5xx internals", () => {
    expect(messageFromError(apiError(500, "pq: syntax error"))).toMatch(/trouble/);
  });
  it("handles rate limiting", () => {
    expect(messageFromError(apiError(429, "x", { retryAfterSeconds: 12 }))).toBe(
      "Too many attempts. Please wait 12 seconds and try again."
    );
    expect(messageFromError(apiError(429, "x", { retryAfterSeconds: 1 }))).toContain("1 second and");
    expect(messageFromError(apiError(429, "x"))).toMatch(/Too many/);
  });
  it("handles size, generic statuses and fallbacks", () => {
    expect(messageFromError(apiError(413, "big"))).toMatch(/5 MB/);
    expect(messageFromError(apiError(404, "API request failed (404)"))).toMatch(/could not find/);
    expect(messageFromError(apiError(400, "API request failed (400)"), "custom")).toBe("custom");
    expect(messageFromError(null, "custom")).toBe("custom");
    expect(messageFromError("boom", "custom")).toBe("custom");
  });
  it("maps network failures and timeouts", () => {
    expect(messageFromError(new TypeError("Network request failed"))).toMatch(/reach the server/);
    const timeout = Object.assign(new Error("t"), { name: "TimeoutError" });
    expect(messageFromError(timeout)).toMatch(/timed out/);
  });
  it("exposes status helpers", () => {
    expect(errorStatus(apiError(404, "x"))).toBe(404);
    expect(errorStatus(new Error("x"))).toBeNull();
    expect(isStatus(apiError(404, "x"), 404, 403)).toBe(true);
    expect(isStatus(apiError(500, "x"), 404)).toBe(false);
  });
});

describe("parseRetryAfter", () => {
  it("parses seconds and dates", () => {
    expect(parseRetryAfter("30")).toBe(30);
    expect(parseRetryAfter(null)).toBeNull();
    expect(parseRetryAfter("garbage")).toBeNull();
    const now = Date.parse("2026-01-01T00:00:00Z");
    expect(parseRetryAfter("Thu, 01 Jan 2026 00:00:10 GMT", now)).toBe(10);
    expect(parseRetryAfter("Wed, 31 Dec 2025 23:00:00 GMT", now)).toBe(0);
  });
});
