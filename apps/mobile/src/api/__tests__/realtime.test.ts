import { RealtimeClient, backoffDelay, parseRealtimeFrame, type SocketLike } from "../realtime";

class FakeSocket implements SocketLike {
  readyState = 0;
  sent: string[] = [];
  closed = false;
  onopen: SocketLike["onopen"] = null;
  onmessage: SocketLike["onmessage"] = null;
  onclose: SocketLike["onclose"] = null;
  onerror: SocketLike["onerror"] = null;
  send(data: string) {
    this.sent.push(data);
  }
  close() {
    this.closed = true;
  }
  open() {
    this.onopen?.({});
  }
  receive(frame: unknown) {
    this.onmessage?.({ data: JSON.stringify(frame) });
  }
  drop(code = 1006) {
    this.onclose?.({ code });
  }
}

describe("backoffDelay", () => {
  it("grows exponentially, is capped and jittered into [50%,100%]", () => {
    expect(backoffDelay(0, 1000, 30000, () => 1)).toBe(1000);
    expect(backoffDelay(3, 1000, 30000, () => 1)).toBe(8000);
    expect(backoffDelay(10, 1000, 30000, () => 1)).toBe(30000);
    expect(backoffDelay(2, 1000, 30000, () => 0)).toBe(2000);
  });
});

describe("parseRealtimeFrame", () => {
  it("accepts known events only", () => {
    expect(parseRealtimeFrame(JSON.stringify({ type: "message.new", data: { id: "1" } }))).toEqual({
      type: "message.new",
      data: { id: "1" }
    });
    expect(parseRealtimeFrame(JSON.stringify({ type: "ready" }))).toBeNull();
    expect(parseRealtimeFrame(JSON.stringify({ type: "evil", data: {} }))).toBeNull();
    expect(parseRealtimeFrame("not json")).toBeNull();
    expect(parseRealtimeFrame(42)).toBeNull();
  });
});

describe("RealtimeClient", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  function make(overrides: Partial<ConstructorParameters<typeof RealtimeClient>[0]> = {}) {
    const sockets: FakeSocket[] = [];
    const onEvent = jest.fn();
    const onReconnect = jest.fn();
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken: () => "tok",
      onEvent,
      onReconnect,
      createSocket: () => {
        const s = new FakeSocket();
        sockets.push(s);
        return s;
      },
      random: () => 1,
      ...overrides
    });
    return { client, sockets, onEvent, onReconnect };
  }

  it("sends the auth frame first and delivers events after ready", () => {
    const { client, sockets, onEvent } = make();
    client.start();
    sockets[0].open();
    expect(JSON.parse(sockets[0].sent[0])).toEqual({ type: "auth", token: "tok" });
    sockets[0].receive({ type: "ready" });
    expect(client.getStatus()).toBe("open");
    sockets[0].receive({ type: "match.removed", data: { matchId: "m", conversationId: "c" } });
    expect(onEvent).toHaveBeenCalledWith({ type: "match.removed", data: { matchId: "m", conversationId: "c" } });
  });

  it("reconnects with exponential backoff and signals onReconnect only after a lost session", () => {
    const { client, sockets, onReconnect } = make();
    client.start();
    sockets[0].open();
    sockets[0].receive({ type: "ready" });
    expect(onReconnect).not.toHaveBeenCalled();

    sockets[0].drop();
    jest.advanceTimersByTime(999);
    expect(sockets).toHaveLength(1);
    jest.advanceTimersByTime(1);
    expect(sockets).toHaveLength(2);

    sockets[1].drop(); // never became ready: next delay doubles
    jest.advanceTimersByTime(1999);
    expect(sockets).toHaveLength(2);
    jest.advanceTimersByTime(1);
    expect(sockets).toHaveLength(3);

    sockets[2].open();
    sockets[2].receive({ type: "ready" });
    expect(onReconnect).toHaveBeenCalledTimes(1);
  });

  it("refreshes the session after close code 4401 before retrying", async () => {
    const refreshSession = jest.fn(async () => true);
    const { client, sockets } = make({ refreshSession });
    client.start();
    sockets[0].open();
    sockets[0].drop(4401);
    await Promise.resolve();
    await Promise.resolve();
    expect(refreshSession).toHaveBeenCalled();
    jest.advanceTimersByTime(1000);
    expect(sockets).toHaveLength(2);
  });

  it("stop() prevents reconnects", () => {
    const { client, sockets } = make();
    client.start();
    sockets[0].open();
    client.stop();
    jest.advanceTimersByTime(60_000);
    expect(sockets).toHaveLength(1);
    expect(client.getStatus()).toBe("idle");
  });
});
