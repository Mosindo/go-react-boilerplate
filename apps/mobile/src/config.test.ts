import { resolveApiBaseUrl, toWebSocketUrl } from "./config";

describe("config", () => {
  it("prefers the explicit URL and strips trailing slashes", () => {
    expect(resolveApiBaseUrl("http://192.168.1.20:18080//", "ios", false)).toBe(
      "http://192.168.1.20:18080"
    );
  });

  it("falls back to emulator addresses in development only", () => {
    expect(resolveApiBaseUrl(undefined, "android", true)).toBe("http://10.0.2.2:18080");
    expect(resolveApiBaseUrl("", "ios", true)).toBe("http://localhost:18080");
    expect(resolveApiBaseUrl(undefined, "ios", false)).toBe("");
  });

  it("derives the websocket URL", () => {
    expect(toWebSocketUrl("http://host:18080")).toBe("ws://host:18080/ws");
    expect(toWebSocketUrl("https://api.example.com")).toBe("wss://api.example.com/ws");
  });
});
