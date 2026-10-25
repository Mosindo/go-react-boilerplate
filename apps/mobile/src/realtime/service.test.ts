import { parseServerFrame, toWebSocketUrl } from "./protocol";
import { AUTH_FAILURE_CLOSE_CODE, PING_INTERVAL_MS, RealtimeService, type SocketLike } from "./service";

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
  receive(frame: object) {
    this.onmessage?.({ data: JSON.stringify(frame) });
  }
  drop(code = 1006) {
    this.onclose?.({ code });
  }
}

function setup() {
  const sockets: FakeSocket[] = [];
  const getToken = jest.fn<Promise<string | null>, [boolean]>(() => Promise.resolve("tok"));
  const onEvent = jest.fn();
  const onResync = jest.fn();
  const service = new RealtimeService({
    url: "ws://api.test/ws",
    getToken,
    onEvent,
    onResync,
    random: () => 1,
    createSocket: () => {
      const socket = new FakeSocket();
      sockets.push(socket);
      return socket;
    }
  });
  return { service, sockets, getToken, onEvent, onResync };
}

const flush = () => jest.advanceTimersByTimeAsync(0);

describe("RealtimeService", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it("authenticates, forwards events and pings every 25 seconds", async () => {
    const { service, sockets, onEvent, onResync } = setup();
    service.start();
    await flush();
    const socket = sockets[0];
    socket.open();
    expect(JSON.parse(socket.sent[0] ?? "")).toEqual({ type: "auth", token: "tok" });

    socket.receive({ type: "ready" });
    socket.receive({ type: "message", data: { id: "m1" } });
    expect(onEvent).toHaveBeenCalledWith({ type: "message", data: { id: "m1" } });
    expect(onResync).not.toHaveBeenCalled();

    await jest.advanceTimersByTimeAsync(PING_INTERVAL_MS);
    expect(socket.sent.map((frame) => (JSON.parse(frame) as { type: string }).type)).toEqual(["auth", "ping"]);
    service.stop();
  });

  it("reconnects with exponential backoff and resyncs REST after the connection returns", async () => {
    const { service, sockets, onResync } = setup();
    service.start();
    await flush();
    sockets[0]?.receive({ type: "ready" });

    sockets[0]?.drop();
    await jest.advanceTimersByTimeAsync(999);
    expect(sockets).toHaveLength(1);
    await jest.advanceTimersByTimeAsync(1);
    expect(sockets).toHaveLength(2);

    sockets[1]?.drop();
    await jest.advanceTimersByTimeAsync(1_999);
    expect(sockets).toHaveLength(2);
    await jest.advanceTimersByTimeAsync(1);
    expect(sockets).toHaveLength(3);

    sockets[2]?.receive({ type: "ready" });
    expect(onResync).toHaveBeenCalledTimes(1);
    service.stop();
  });

  it("asks for a fresh token after an auth failure (4401)", async () => {
    const { service, sockets, getToken } = setup();
    service.start();
    await flush();
    sockets[0]?.drop(AUTH_FAILURE_CLOSE_CODE);
    await jest.advanceTimersByTimeAsync(1_000);
    expect(getToken).toHaveBeenNthCalledWith(1, false);
    expect(getToken).toHaveBeenNthCalledWith(2, true);
    service.stop();
  });

  it("closes a dead connection when a ping gets no answer", async () => {
    const { service, sockets } = setup();
    service.start();
    await flush();
    sockets[0]?.receive({ type: "ready" });
    await jest.advanceTimersByTimeAsync(PING_INTERVAL_MS * 2);
    expect(sockets[0]?.closed).toBe(true);
    service.stop();
  });

  it("stops reconnecting once stopped", async () => {
    const { service, sockets } = setup();
    service.start();
    await flush();
    service.stop();
    await jest.advanceTimersByTimeAsync(60_000);
    expect(sockets).toHaveLength(1);
  });
});

describe("protocol", () => {
  it("parses frames and rejects junk", () => {
    expect(parseServerFrame('{"type":"ready"}')).toEqual({ kind: "ready" });
    expect(parseServerFrame('{"type":"pong"}')).toEqual({ kind: "pong" });
    expect(parseServerFrame('{"type":"read","data":{"conversationId":"c"}}')).toMatchObject({ kind: "event" });
    expect(parseServerFrame('{"type":"unknown","data":{}}')).toBeNull();
    expect(parseServerFrame("not json")).toBeNull();
    expect(parseServerFrame(42)).toBeNull();
  });

  it("derives the websocket url", () => {
    expect(toWebSocketUrl("http://10.0.0.2:18080")).toBe("ws://10.0.0.2:18080/ws");
    expect(toWebSocketUrl("https://api.example.org")).toBe("wss://api.example.org/ws");
  });
});
