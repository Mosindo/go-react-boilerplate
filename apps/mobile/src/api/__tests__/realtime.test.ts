import { PING_INTERVAL_MS, RealtimeClient, backoffDelayMs, parseServerFrame, type SocketLike } from "../realtime";
import { toWebSocketUrl } from "../config";

jest.mock("react-native", () => ({ Platform: { OS: "ios" } }));

class FakeSocket implements SocketLike {
  readyState = 1;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  sent: string[] = [];
  closed = false;
  send(data: string) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
    this.onclose?.();
  }
  receive(frame: object) {
    this.onmessage?.({ data: JSON.stringify(frame) });
  }
}

const message = {
  id: "m1",
  conversationId: "c1",
  senderId: "u1",
  body: "hi",
  createdAt: "2026-01-01T00:00:00Z",
  readAt: null
};

describe("parseServerFrame", () => {
  it("parses control frames and valid events", () => {
    expect(parseServerFrame('{"type":"ready"}')).toEqual({ kind: "ready" });
    expect(parseServerFrame('{"type":"pong"}')).toEqual({ kind: "pong" });
    expect(parseServerFrame(JSON.stringify({ type: "message.new", data: message }))).toEqual({
      kind: "event",
      event: { type: "message.new", data: message }
    });
    expect(parseServerFrame('{"type":"messages.read","data":{"conversationId":"c","readerId":"r"}}')).toEqual({
      kind: "event",
      event: { type: "messages.read", data: { conversationId: "c", readerId: "r" } }
    });
    expect(parseServerFrame('{"type":"match.removed","data":{"matchId":"m","conversationId":"c"}}')).toMatchObject({
      kind: "event"
    });
  });

  it("rejects malformed or unknown frames", () => {
    expect(parseServerFrame("not json")).toBeNull();
    expect(parseServerFrame('{"type":"message.new","data":{"id":1}}')).toBeNull();
    expect(parseServerFrame('{"type":"mystery","data":{}}')).toBeNull();
    expect(parseServerFrame("[]")).toBeNull();
    expect(parseServerFrame('{"type":"notification.new","data":{"id":"n"}}')).toBeNull();
    expect(parseServerFrame('{"type":"match.new","data":{"matchId":"m"}}')).toBeNull();
  });
});

describe("backoffDelayMs", () => {
  it("grows exponentially and is capped at 30 seconds", () => {
    const noJitter = () => 1;
    expect(backoffDelayMs(0, noJitter)).toBe(1000);
    expect(backoffDelayMs(1, noJitter)).toBe(2000);
    expect(backoffDelayMs(3, noJitter)).toBe(8000);
    expect(backoffDelayMs(10, noJitter)).toBe(30000);
    expect(backoffDelayMs(0, () => 0)).toBe(750);
  });
});

describe("toWebSocketUrl", () => {
  it("swaps the scheme", () => {
    expect(toWebSocketUrl("http://192.168.1.2:18080", "/ws")).toBe("ws://192.168.1.2:18080/ws");
    expect(toWebSocketUrl("https://api.example.com", "/ws")).toBe("wss://api.example.com/ws");
  });
});

describe("RealtimeClient", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  function setup() {
    const sockets: FakeSocket[] = [];
    const events: unknown[] = [];
    const onReconnected = jest.fn();
    const client = new RealtimeClient({
      url: "ws://api/ws",
      getToken: async () => "tok",
      onEvent: (event) => events.push(event),
      onReconnected,
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket;
      }
    });
    return { client, sockets, events, onReconnected };
  }

  async function flush() {
    await Promise.resolve();
    await Promise.resolve();
  }

  it("authenticates on open, dispatches events and pings every 25 seconds", async () => {
    const { client, sockets, events } = setup();
    client.start();
    await flush();
    sockets[0].onopen?.();
    expect(JSON.parse(sockets[0].sent[0])).toEqual({ type: "auth", token: "tok" });
    sockets[0].receive({ type: "ready" });
    sockets[0].receive({ type: "message.new", data: message });
    expect(events).toEqual([{ type: "message.new", data: message }]);

    jest.advanceTimersByTime(PING_INTERVAL_MS);
    expect(JSON.parse(sockets[0].sent[1])).toEqual({ type: "ping" });
    sockets[0].receive({ type: "pong" });
    jest.advanceTimersByTime(PING_INTERVAL_MS);
    expect(sockets[0].sent.filter((frame) => frame.includes("ping"))).toHaveLength(2);
    client.stop();
  });

  it("reconnects with backoff and asks callers to refetch after the second ready", async () => {
    const { client, sockets, onReconnected } = setup();
    client.start();
    await flush();
    sockets[0].onopen?.();
    sockets[0].receive({ type: "ready" });
    expect(onReconnected).not.toHaveBeenCalled();

    sockets[0].onclose?.();
    jest.advanceTimersByTime(1000);
    await flush();
    expect(sockets).toHaveLength(2);
    sockets[1].onopen?.();
    sockets[1].receive({ type: "ready" });
    expect(onReconnected).toHaveBeenCalledTimes(1);
    client.stop();
  });

  it("closes a connection that stops answering pings", async () => {
    const { client, sockets } = setup();
    client.start();
    await flush();
    sockets[0].onopen?.();
    sockets[0].receive({ type: "ready" });
    jest.advanceTimersByTime(PING_INTERVAL_MS + 10_000);
    expect(sockets[0].closed).toBe(true);
    client.stop();
  });

  it("stop() closes the socket and prevents reconnects", async () => {
    const { client, sockets } = setup();
    client.start();
    await flush();
    client.stop();
    expect(sockets[0].closed).toBe(true);
    jest.advanceTimersByTime(60_000);
    await flush();
    expect(sockets).toHaveLength(1);
  });

  it("retries later when no token is available", async () => {
    const sockets: FakeSocket[] = [];
    let token: string | null = null;
    const client = new RealtimeClient({
      url: "ws://api/ws",
      getToken: async () => token,
      onEvent: () => undefined,
      onReconnected: () => undefined,
      createSocket: () => {
        const socket = new FakeSocket();
        sockets.push(socket);
        return socket;
      }
    });
    client.start();
    await flush();
    expect(sockets).toHaveLength(0);
    token = "tok";
    jest.advanceTimersByTime(1000);
    await flush();
    expect(sockets).toHaveLength(1);
    client.stop();
  });
});
