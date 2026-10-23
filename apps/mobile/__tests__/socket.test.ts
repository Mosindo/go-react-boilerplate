import { toWebSocketUrl, resolveBaseUrl, normalizeBaseUrl } from "../src/api/baseUrl";
import {
  BACKOFF_MAX_MS,
  PING_INTERVAL_MS,
  RealtimeClient,
  backoffDelay
} from "../src/realtime/socket";

class FakeSocket {
  static instances: FakeSocket[] = [];
  onopen: ((event: unknown) => void) | null = null;
  onmessage: ((event: { data: unknown }) => void) | null = null;
  onclose: ((event: unknown) => void) | null = null;
  onerror: ((event: unknown) => void) | null = null;
  sent: string[] = [];
  closed = false;
  constructor(public url: string) {
    FakeSocket.instances.push(this);
  }
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
  drop() {
    this.onclose?.({});
  }
}

const flush = async () => {
  for (let i = 0; i < 5; i += 1) await Promise.resolve();
};

describe("backoff", () => {
  it("doubles from 1s and caps at 30s", () => {
    const noJitter = () => 0;
    expect([0, 1, 2, 3, 4, 5, 6, 10].map((n) => backoffDelay(n, noJitter))).toEqual([
      1000, 2000, 4000, 8000, 16000, 30000, 30000, 30000
    ]);
  });

  it("adds at most 25% jitter and never exceeds the cap", () => {
    expect(backoffDelay(1, () => 1)).toBe(2500);
    expect(backoffDelay(9, () => 1)).toBe(BACKOFF_MAX_MS);
  });
});

describe("RealtimeClient", () => {
  beforeEach(() => {
    jest.useFakeTimers();
    FakeSocket.instances = [];
  });
  afterEach(() => jest.useRealTimers());

  function build() {
    const onMessage = jest.fn();
    const onReconnect = jest.fn();
    const onStatus = jest.fn();
    const getToken = jest.fn(async () => "tok");
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken,
      onMessage,
      onReconnect,
      onStatus,
      random: () => 0,
      createSocket: (url) => new FakeSocket(url)
    });
    return { client, onMessage, onReconnect, onStatus, getToken };
  }

  it("sends the auth frame first (in onopen), then forwards events but not control frames", async () => {
    const { client, onMessage } = build();
    client.start();
    await flush();
    const socket = FakeSocket.instances[0];
    expect(socket.url).toBe("ws://x/ws");
    socket.open();
    expect(JSON.parse(socket.sent[0])).toEqual({ type: "auth", token: "tok" });
    socket.receive({ type: "ready" });
    socket.receive({ type: "pong" });
    socket.receive({ type: "message.new", data: { id: "1" } });
    expect(onMessage).toHaveBeenCalledTimes(1);
    client.stop();
  });

  it("pings periodically once ready", async () => {
    const { client } = build();
    client.start();
    await flush();
    const socket = FakeSocket.instances[0];
    socket.open();
    socket.receive({ type: "ready" });
    jest.advanceTimersByTime(PING_INTERVAL_MS);
    expect(socket.sent.map((s) => JSON.parse(s).type)).toEqual(["auth", "ping"]);
    client.stop();
  });

  it("reconnects with backoff, re-authenticates and signals a refetch only after the first ready", async () => {
    const { client, onReconnect, getToken } = build();
    client.start();
    await flush();
    const first = FakeSocket.instances[0];
    first.open();
    first.receive({ type: "ready" });
    expect(onReconnect).not.toHaveBeenCalled();

    first.drop();
    jest.advanceTimersByTime(999);
    expect(FakeSocket.instances).toHaveLength(1);
    jest.advanceTimersByTime(1);
    await flush();
    expect(FakeSocket.instances).toHaveLength(2);
    expect(getToken).toHaveBeenCalledTimes(2);

    const second = FakeSocket.instances[1];
    second.open();
    second.receive({ type: "ready" });
    expect(onReconnect).toHaveBeenCalledTimes(1);

    // a failed attempt backs off 1s -> 2s (attempt resets after ready)
    second.drop();
    jest.advanceTimersByTime(1000);
    await flush();
    FakeSocket.instances[2].drop();
    jest.advanceTimersByTime(1999);
    expect(FakeSocket.instances).toHaveLength(3);
    jest.advanceTimersByTime(1);
    await flush();
    expect(FakeSocket.instances).toHaveLength(4);
    client.stop();
  });

  it("does not reconnect after stop()", async () => {
    const { client } = build();
    client.start();
    await flush();
    const socket = FakeSocket.instances[0];
    client.stop();
    expect(socket.closed).toBe(true);
    jest.advanceTimersByTime(60_000);
    await flush();
    expect(FakeSocket.instances).toHaveLength(1);
  });

  it("retries later when there is no token", async () => {
    const onStatus = jest.fn();
    let token: string | null = null;
    const client = new RealtimeClient({
      url: "ws://x/ws",
      getToken: async () => token,
      onMessage: jest.fn(),
      onStatus,
      random: () => 0,
      createSocket: (url) => new FakeSocket(url)
    });
    client.start();
    await flush();
    expect(FakeSocket.instances).toHaveLength(0);
    token = "tok";
    jest.advanceTimersByTime(1000);
    await flush();
    expect(FakeSocket.instances).toHaveLength(1);
    client.stop();
  });
});

describe("base url", () => {
  it("prefers EXPO_PUBLIC_API_URL and trims trailing slashes", () => {
    expect(
      resolveBaseUrl({ explicit: " http://192.168.1.5:18080/ ", isDev: true, os: "ios" })
    ).toBe("http://192.168.1.5:18080");
    expect(normalizeBaseUrl("http://a///")).toBe("http://a");
  });

  it("falls back to the dev host on port 18080", () => {
    expect(resolveBaseUrl({ explicit: undefined, isDev: true, os: "android" })).toBe(
      "http://10.0.2.2:18080"
    );
    expect(resolveBaseUrl({ explicit: "", isDev: true, os: "ios" })).toBe("http://localhost:18080");
    expect(resolveBaseUrl({ explicit: undefined, isDev: false, os: "ios" })).toBe("");
  });

  it("derives the websocket url", () => {
    expect(toWebSocketUrl("http://h:18080")).toBe("ws://h:18080/ws");
    expect(toWebSocketUrl("https://api.example.com/")).toBe("wss://api.example.com/ws");
  });
});
