import { backoffDelay, RealtimeClient, type SocketLike } from "../src/realtime/socket";
import type { RealtimeEvent } from "../src/realtime/cache";

class FakeSocket implements SocketLike {
  onopen: (() => void) | null = null;
  onmessage: ((e: { data: unknown }) => void) | null = null;
  onclose: ((e: { code?: number }) => void) | null = null;
  onerror: (() => void) | null = null;
  sent: string[] = [];
  closed = false;
  send(data: string) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
  }
}

const flush = () => new Promise((r) => setImmediate(r));

describe("backoffDelay", () => {
  it("grows exponentially up to a cap", () => {
    expect([0, 1, 2, 3].map((n) => backoffDelay(n))).toEqual([1000, 2000, 4000, 8000]);
    expect(backoffDelay(20)).toBe(30_000);
  });
});

describe("RealtimeClient", () => {
  it("authenticates, becomes ready and forwards events", async () => {
    const sockets: FakeSocket[] = [];
    const events: RealtimeEvent[] = [];
    const statuses: string[] = [];
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken: async () => "tok",
      onEvent: (e) => events.push(e),
      onStatus: (s) => statuses.push(s),
      createSocket: () => {
        const s = new FakeSocket();
        sockets.push(s);
        return s;
      },
      pingIntervalMs: 1_000_000
    });
    client.start();
    await flush();
    const socket = sockets[0] as FakeSocket;
    socket.onopen?.();
    expect(JSON.parse(socket.sent[0] as string)).toEqual({ type: "auth", token: "tok" });
    socket.onmessage?.({ data: JSON.stringify({ type: "ready" }) });
    expect(statuses).toContain("open");
    socket.onmessage?.({ data: JSON.stringify({ type: "pong" }) });
    socket.onmessage?.({
      data: JSON.stringify({ type: "message", data: { id: "m", conversationId: "c", senderId: "s", body: "hi", createdAt: "t", readAt: null } })
    });
    expect(events).toHaveLength(1);
    client.stop();
    expect(socket.closed).toBe(true);
  });

  it("reconnects with backoff after a close and stops cleanly", async () => {
    jest.useFakeTimers();
    const sockets: FakeSocket[] = [];
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken: async () => "tok",
      onEvent: () => undefined,
      createSocket: () => {
        const s = new FakeSocket();
        sockets.push(s);
        return s;
      }
    });
    client.start();
    await Promise.resolve();
    await Promise.resolve();
    expect(sockets).toHaveLength(1);
    sockets[0]?.onclose?.({ code: 1006 });
    jest.advanceTimersByTime(999);
    expect(sockets).toHaveLength(1);
    jest.advanceTimersByTime(2);
    await Promise.resolve();
    await Promise.resolve();
    expect(sockets).toHaveLength(2);
    client.stop();
    sockets[1]?.onclose?.({ code: 1006 });
    jest.advanceTimersByTime(60_000);
    expect(sockets).toHaveLength(2);
    jest.useRealTimers();
  });

  it("refreshes credentials when the server rejects the token (4401)", async () => {
    jest.useFakeTimers();
    const refresh = jest.fn(async () => "new");
    const sockets: FakeSocket[] = [];
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken: async () => "tok",
      refreshToken: refresh,
      onEvent: () => undefined,
      createSocket: () => {
        const s = new FakeSocket();
        sockets.push(s);
        return s;
      }
    });
    client.start();
    await Promise.resolve();
    await Promise.resolve();
    sockets[0]?.onclose?.({ code: 4401 });
    expect(refresh).toHaveBeenCalledTimes(1);
    client.stop();
    jest.useRealTimers();
  });
});
