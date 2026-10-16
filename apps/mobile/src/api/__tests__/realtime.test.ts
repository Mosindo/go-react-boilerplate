import { describe, expect, it, vi } from "vitest";

// realtime.ts pulls in the API client (React Native); only the pure helpers are under test here.
vi.mock("../client", () => ({ API_BASE_URL: "http://api.test" }));
vi.mock("../platform", () => ({ requestRealtimeTicket: vi.fn() }));

import { parseEvent, reconnectDelay, toSocketUrl } from "../realtime";

describe("realtime helpers", () => {
  it("derives a ws(s) URL and encodes the ticket", () => {
    expect(toSocketUrl("http://192.168.1.10:18080", "a b")).toBe("ws://192.168.1.10:18080/realtime?ticket=a%20b");
    expect(toSocketUrl("https://api.example.com", "t")).toBe("wss://api.example.com/realtime?ticket=t");
  });

  it("backs off exponentially up to a cap", () => {
    expect(reconnectDelay(0)).toBe(1000);
    expect(reconnectDelay(3)).toBe(8000);
    expect(reconnectDelay(20)).toBe(30000);
  });

  it("accepts known events and ignores everything else", () => {
    expect(parseEvent('{"type":"match.new","data":{"matchId":"m","userId":"u"}}')?.type).toBe("match.new");
    expect(parseEvent('{"type":"unknown"}')).toBeNull();
    expect(parseEvent("not json")).toBeNull();
  });
});
