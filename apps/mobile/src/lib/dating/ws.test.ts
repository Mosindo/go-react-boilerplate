import { describe, expect, it } from "vitest";
import { BACKOFF_MAX_MS, buildWsUrlFromBase, computeBackoffMs, parseRealtimeEvent } from "./ws";

describe("buildWsUrlFromBase", () => {
  it("converts http to ws and https to wss", () => {
    expect(buildWsUrlFromBase("http://192.168.1.2:18080", "abc")).toBe(
      "ws://192.168.1.2:18080/ws?ticket=abc"
    );
    expect(buildWsUrlFromBase("https://api.alba.app/", "a b")).toBe(
      "wss://api.alba.app/ws?ticket=a%20b"
    );
  });
});

describe("computeBackoffMs", () => {
  it("grows exponentially inside the jitter window and caps at 30 s", () => {
    expect(computeBackoffMs(0, () => 1)).toBe(1000);
    expect(computeBackoffMs(0, () => 0)).toBe(500);
    expect(computeBackoffMs(3, () => 1)).toBe(8000);
    expect(computeBackoffMs(10, () => 1)).toBe(BACKOFF_MAX_MS);
    expect(computeBackoffMs(50, () => 0)).toBe(BACKOFF_MAX_MS / 2);
  });
});

const message = {
  id: "m1",
  conversationId: "c1",
  senderId: "u",
  body: "hi",
  createdAt: "2026-01-01T00:00:00Z",
  readAt: null
};

describe("parseRealtimeEvent", () => {
  it("parses known events", () => {
    expect(parseRealtimeEvent(JSON.stringify({ type: "message.new", data: { message } }))).toEqual({
      type: "message.new",
      data: { message }
    });
    expect(
      parseRealtimeEvent({
        type: "conversation.read",
        data: { conversationId: "c", readAt: "t" }
      })
    ).toEqual({
      type: "conversation.read",
      data: { conversationId: "c", readAt: "t" }
    });
    expect(
      parseRealtimeEvent({
        type: "match.removed",
        data: { conversationId: "c" }
      })
    ).not.toBeNull();
  });

  it("parses match.new and notification.new", () => {
    const conversation = {
      id: "c1",
      matchId: "m",
      matchedAt: "t",
      user: { userId: "u", firstName: "A", age: null, photo: null },
      lastMessage: null,
      unreadCount: 0,
      updatedAt: "t"
    };
    expect(parseRealtimeEvent({ type: "match.new", data: { conversation } })?.type).toBe(
      "match.new"
    );
    const notification = {
      id: "n",
      type: "match",
      title: "t",
      body: "b",
      data: {},
      isRead: false,
      createdAt: "t"
    };
    expect(parseRealtimeEvent({ type: "notification.new", data: { notification } })?.type).toBe(
      "notification.new"
    );
  });

  it("rejects malformed and unknown frames", () => {
    expect(parseRealtimeEvent("not json")).toBeNull();
    expect(parseRealtimeEvent(null)).toBeNull();
    expect(parseRealtimeEvent({ type: "message.new", data: { message: { id: 1 } } })).toBeNull();
    expect(parseRealtimeEvent({ type: "future.thing", data: {} })).toBeNull();
    expect(
      parseRealtimeEvent({
        type: "match.new",
        data: { conversation: { id: "x" } }
      })
    ).toBeNull();
  });
});
