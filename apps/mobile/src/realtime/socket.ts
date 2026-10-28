import { parseRealtimeEvent, type RealtimeEvent } from "./cache";

export type SocketStatus = "idle" | "connecting" | "open" | "closed";

export type SocketLike = {
  onopen: (() => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: ((event: { code?: number }) => void) | null;
  onerror: (() => void) | null;
  send: (data: string) => void;
  close: () => void;
};

export type RealtimeClientOptions = {
  url: string;
  getToken: () => Promise<string | null>;
  /** Called when the server rejects our token (close 4401); should refresh credentials. */
  refreshToken?: () => Promise<string | null>;
  onEvent: (event: RealtimeEvent) => void;
  onStatus?: (status: SocketStatus) => void;
  createSocket?: (url: string) => SocketLike;
  pingIntervalMs?: number;
  setTimer?: (fn: () => void, ms: number) => ReturnType<typeof setTimeout>;
  clearTimer?: (id: ReturnType<typeof setTimeout>) => void;
};

const AUTH_CLOSE_CODE = 4401;

export function backoffDelay(attempt: number, base = 1000, max = 30_000): number {
  return Math.min(max, base * 2 ** Math.max(0, attempt));
}

export class RealtimeClient {
  private socket: SocketLike | null = null;
  private attempt = 0;
  private running = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private generation = 0;

  constructor(private readonly options: RealtimeClientOptions) {}

  start(): void {
    if (this.running) return;
    this.running = true;
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.generation += 1;
    this.teardown();
    this.options.onStatus?.("idle");
  }

  private teardown(): void {
    if (this.reconnectTimer) {
      (this.options.clearTimer ?? clearTimeout)(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
    if (this.socket) {
      const socket = this.socket;
      this.socket = null;
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
      try {
        socket.close();
      } catch {
        // already closed
      }
    }
  }

  private async connect(): Promise<void> {
    const generation = ++this.generation;
    this.options.onStatus?.("connecting");
    const token = await this.options.getToken();
    if (!this.running || generation !== this.generation) return;
    if (!token) {
      this.scheduleReconnect();
      return;
    }
    const socket = (this.options.createSocket ?? ((u) => new WebSocket(u) as unknown as SocketLike))(this.options.url);
    this.socket = socket;
    let ready = false;

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (event) => {
      if (typeof event.data !== "string") return;
      try {
        const frame = JSON.parse(event.data) as { type?: string };
        if (frame.type === "ready") {
          ready = true;
          this.attempt = 0;
          this.options.onStatus?.("open");
          this.startPing(socket);
          return;
        }
        if (frame.type === "pong") return;
      } catch {
        return;
      }
      const parsed = parseRealtimeEvent(event.data);
      if (parsed) this.options.onEvent(parsed);
    };
    socket.onerror = () => {
      // close handler drives reconnection
    };
    socket.onclose = (event) => {
      if (generation !== this.generation) return;
      this.teardown();
      this.options.onStatus?.("closed");
      if (!this.running) return;
      if (event.code === AUTH_CLOSE_CODE && !ready && this.options.refreshToken) {
        void this.options.refreshToken().finally(() => this.scheduleReconnect());
        return;
      }
      this.scheduleReconnect();
    };
  }

  private startPing(socket: SocketLike): void {
    if (this.pingTimer) clearInterval(this.pingTimer);
    this.pingTimer = setInterval(() => {
      try {
        socket.send(JSON.stringify({ type: "ping" }));
      } catch {
        // the close handler will reconnect
      }
    }, this.options.pingIntervalMs ?? 25_000);
  }

  private scheduleReconnect(): void {
    if (!this.running) return;
    const delay = backoffDelay(this.attempt++);
    this.reconnectTimer = (this.options.setTimer ?? setTimeout)(() => {
      this.reconnectTimer = null;
      void this.connect();
    }, delay);
  }
}
