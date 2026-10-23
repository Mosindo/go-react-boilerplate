export type SocketStatus = "idle" | "connecting" | "open" | "closed";

export const BACKOFF_BASE_MS = 1000;
export const BACKOFF_MAX_MS = 30_000;
export const PING_INTERVAL_MS = 25_000;

/** Exponential backoff with up to 25% jitter, never above 30 s. `attempt` starts at 0. */
export function backoffDelay(attempt: number, random: () => number = Math.random): number {
  const exp = BACKOFF_BASE_MS * 2 ** Math.max(0, attempt);
  return Math.round(Math.min(BACKOFF_MAX_MS, exp * (1 + random() * 0.25)));
}

type MinimalSocket = {
  onopen: ((event: unknown) => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: ((event: unknown) => void) | null;
  onerror: ((event: unknown) => void) | null;
  send(data: string): void;
  close(): void;
};

export type SocketFactory = (url: string) => MinimalSocket;

export type RealtimeOptions = {
  url: string;
  /** Returns a fresh access token (refreshing if needed) or null when signed out. */
  getToken: () => Promise<string | null>;
  onMessage: (raw: unknown) => void;
  onStatus?: (status: SocketStatus) => void;
  /** Fired on every `ready` after the first one: the cache may be stale. */
  onReconnect?: () => void;
  createSocket?: SocketFactory;
  random?: () => number;
};

/**
 * WebSocket wrapper: auth frame is the first frame (sent in onopen), ping keep-alive,
 * exponential backoff reconnect, refetch signal on reconnect.
 */
export class RealtimeClient {
  private socket: MinimalSocket | null = null;
  private attempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private stopped = true;
  private hasBeenReady = false;
  private generation = 0;

  constructor(private readonly options: RealtimeOptions) {}

  start(): void {
    if (!this.stopped) return;
    this.stopped = false;
    this.attempt = 0;
    this.hasBeenReady = false;
    void this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.generation += 1;
    this.clearTimers();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
      try {
        socket.close();
      } catch {
        // already closed
      }
    }
    this.options.onStatus?.("closed");
  }

  /** Forces an immediate reconnect attempt (e.g. app returned to foreground). */
  reconnectNow(): void {
    if (this.stopped || this.socket) return;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
    void this.connect();
  }

  private clearTimers() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.pingTimer) clearInterval(this.pingTimer);
    this.reconnectTimer = null;
    this.pingTimer = null;
  }

  private async connect(): Promise<void> {
    const generation = ++this.generation;
    this.options.onStatus?.("connecting");
    let token: string | null = null;
    try {
      token = await this.options.getToken();
    } catch {
      token = null;
    }
    if (this.stopped || generation !== this.generation) return;
    if (!token) {
      this.scheduleReconnect();
      return;
    }
    const factory: SocketFactory =
      this.options.createSocket ?? ((url) => new WebSocket(url) as unknown as MinimalSocket);
    const socket = factory(this.options.url);
    this.socket = socket;

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (event) => {
      const raw = event.data;
      if (typeof raw === "string") {
        try {
          const frame = JSON.parse(raw) as { type?: string };
          if (frame.type === "ready") {
            this.attempt = 0;
            this.options.onStatus?.("open");
            this.startPing(socket);
            if (this.hasBeenReady) this.options.onReconnect?.();
            this.hasBeenReady = true;
            return;
          }
          if (frame.type === "pong") return;
        } catch {
          return;
        }
      }
      this.options.onMessage(raw);
    };
    socket.onerror = () => {
      // onclose follows and drives the reconnect
    };
    socket.onclose = () => {
      if (this.socket !== socket) return;
      this.socket = null;
      if (this.pingTimer) clearInterval(this.pingTimer);
      this.pingTimer = null;
      this.options.onStatus?.("closed");
      if (!this.stopped) this.scheduleReconnect();
    };
  }

  private startPing(socket: MinimalSocket) {
    if (this.pingTimer) clearInterval(this.pingTimer);
    this.pingTimer = setInterval(() => {
      try {
        socket.send(JSON.stringify({ type: "ping" }));
      } catch {
        // close handler takes over
      }
    }, PING_INTERVAL_MS);
  }

  private scheduleReconnect() {
    if (this.stopped || this.reconnectTimer) return;
    const delay = backoffDelay(this.attempt, this.options.random);
    this.attempt += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect();
    }, delay);
  }
}
