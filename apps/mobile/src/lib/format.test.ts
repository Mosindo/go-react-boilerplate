import { describe, expect, it } from "vitest";
import { formatDistance, toWebSocketUrl, validateEmail, validatePassword } from "./format";

describe("format helpers", () => {
  it("formats approximate distances", () => {
    expect(formatDistance(undefined)).toBeNull();
    expect(formatDistance(1)).toBe("less than 1 km away");
    expect(formatDistance(15)).toBe("15 km away");
  });
  it("derives the websocket url from the API base", () => {
    expect(toWebSocketUrl("http://192.168.1.10:18080")).toBe("ws://192.168.1.10:18080/ws");
    expect(toWebSocketUrl("https://api.example.com/")).toBe("wss://api.example.com/ws");
  });
  it("validates email and password like the server", () => {
    expect(validateEmail("")).not.toBeNull();
    expect(validateEmail("nope")).not.toBeNull();
    expect(validateEmail(" a@b.co ")).toBeNull();
    expect(validatePassword("short")).not.toBeNull();
    expect(validatePassword("long-enough")).toBeNull();
    expect(validatePassword("é".repeat(37))).not.toBeNull(); // 74 bytes
  });
});
