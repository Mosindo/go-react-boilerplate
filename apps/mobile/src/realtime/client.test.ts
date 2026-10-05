import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RealtimeEvent } from "../api/types";
import { RealtimeClient, type RealtimeStatus, type SocketLike } from "./client";

class FakeSocket implements SocketLike {
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  onclose: ((event: { code: number }) => void) | null = null;
  onerror: (() => void) | null = null;
  sent: string[] = [];
  closed = false;
  send(data: string) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
  }
  open() {
    this.onopen?.();
  }
  receive(event: unknown) {
    this.onmessage?.({ data: JSON.stringify(event) });
  }
  drop(code = 1006) {
    this.onclose?.({ code });
  }
}

type Timer = { fn: () => void; ms: number };

function setup(options: { tokens?: (string | null)[]; refresh?: () => Promise<string | null> } = {}) {
  const sockets: FakeSocket[] = [];
  const timers: Timer[] = [];
  const events: RealtimeEvent[] = [];
  const statuses: RealtimeStatus[] = [];
  const connected: boolean[] = [];
  const tokens = [...(options.tokens ?? ["token-1"])];
  const client = new RealtimeClient({
    url: "ws://api/ws",
    getAccessToken: async () => (tokens.length > 1 ? tokens.shift()! : (tokens[0] ?? null)),
    refreshAccessToken: options.refresh ?? (async () => "refreshed"),
    createSocket: () => {
      const socket = new FakeSocket();
      sockets.push(socket);
      return socket;
    },
    onEvent: (e) => events.push(e),
    onStatus: (s) => statuses.push(s),
    onConnected: (isReconnect) => connected.push(isReconnect),
    random: () => 1,
    setTimer: (fn, ms) => {
      timers.push({ fn, ms });
      return timers.length - 1;
    },
    clearTimer: (handle) => {
      timers[handle as number] = { fn: () => undefined, ms: -1 };
    }
  });
  const flush = () => new Promise((resolve) => setTimeout(resolve, 0));
  return { client, sockets, timers, events, statuses, connected, flush };
}

describe("RealtimeClient", () => {
  beforeEach(() => vi.useRealTimers());

  it("authenticates with the first frame and becomes connected on ready", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].open();
    expect(JSON.parse(t.sockets[0].sent[0])).toEqual({ type: "auth", token: "token-1" });
    expect(t.client.currentStatus).toBe("connecting");

    t.sockets[0].receive({ type: "ready" });
    expect(t.client.currentStatus).toBe("connected");
    expect(t.connected).toEqual([false]);
  });

  it("forwards events but not the ready frame", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].receive({ type: "ready" });
    t.sockets[0].receive({ type: "match.removed", data: { matchId: "m1" } });
    expect(t.events).toEqual([{ type: "match.removed", data: { matchId: "m1" } }]);
  });

  it("ignores malformed frames", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].onmessage?.({ data: "not json" });
    t.sockets[0].onmessage?.({ data: 42 });
    expect(t.events).toEqual([]);
  });

  it("reconnects with exponential backoff and reports the reconnect", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].receive({ type: "ready" });

    t.sockets[0].drop(1006);
    expect(t.client.currentStatus).toBe("reconnecting");
    expect(t.timers.at(-1)?.ms).toBe(1000); // random()=1 → full step

    t.timers.at(-1)!.fn();
    await t.flush();
    t.sockets[1].drop(1006); // fails again before ready
    expect(t.timers.at(-1)?.ms).toBe(2000);
    t.timers.at(-1)!.fn();
    await t.flush();
    t.sockets[2].drop(1006);
    expect(t.timers.at(-1)?.ms).toBe(4000);

    t.timers.at(-1)!.fn();
    await t.flush();
    t.sockets[3].receive({ type: "ready" });
    expect(t.connected).toEqual([false, true]); // the second ready is a reconnect → app refetches
    t.sockets[3].drop();
    expect(t.timers.at(-1)?.ms).toBe(1000); // backoff reset after a successful connection
  });

  it("caps the delay at 30 seconds", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    for (let i = 0; i < 8; i++) {
      t.sockets[i].drop();
      t.timers.at(-1)!.fn();
      await t.flush();
    }
    t.sockets[8].drop();
    expect(t.timers.at(-1)?.ms).toBeLessThanOrEqual(30_000);
  });

  it("refreshes the token when the server rejects it (4401) and reconnects immediately", async () => {
    const refresh = vi.fn(async () => "fresh-token");
    const t = setup({ refresh });
    t.client.start();
    await t.flush();
    t.sockets[0].drop(4401);
    await t.flush();

    expect(refresh).toHaveBeenCalledOnce();
    expect(t.sockets).toHaveLength(2);
    t.sockets[1].open();
    expect(JSON.parse(t.sockets[1].sent[0]).token).toBe("fresh-token");
  });

  it("falls back to a slow retry when the refresh fails", async () => {
    const t = setup({ refresh: async () => null });
    t.client.start();
    await t.flush();
    t.sockets[0].drop(4401);
    await t.flush();
    expect(t.sockets).toHaveLength(1);
    expect(t.timers.length).toBeGreaterThan(0);
  });

  it("stays idle without a token", async () => {
    const t = setup({ tokens: [null] });
    t.client.start();
    await t.flush();
    expect(t.sockets).toHaveLength(0);
    expect(t.client.currentStatus).toBe("idle");
  });

  it("stop() closes the socket and cancels pending reconnects", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].drop();
    t.client.stop();
    t.timers.forEach((timer) => timer.fn()); // even if a timer fires late, nothing reconnects
    await t.flush();
    expect(t.sockets).toHaveLength(1);
    expect(t.client.currentStatus).toBe("idle");
  });

  it("ignores events from a socket that was replaced", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    const old = t.sockets[0];
    old.drop();
    t.timers.at(-1)!.fn();
    await t.flush();
    old.receive({ type: "match.removed", data: { matchId: "stale" } });
    expect(t.events).toEqual([]);
  });

  it("reconnectNow() skips the backoff only when the socket is down", async () => {
    const t = setup();
    t.client.start();
    await t.flush();
    t.sockets[0].receive({ type: "ready" });
    t.client.reconnectNow();
    await t.flush();
    expect(t.sockets).toHaveLength(1); // connected: nothing to do

    t.sockets[0].drop();
    t.client.reconnectNow();
    await t.flush();
    expect(t.sockets).toHaveLength(2);
  });
});
