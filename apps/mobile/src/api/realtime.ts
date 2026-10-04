import type { RealtimeEvent } from "./types";

export type RealtimeStatus = "idle" | "connecting" | "open" | "closed";

/** Minimal WebSocket surface (browser/RN compatible), injectable for tests. */
export type SocketLike = {
  readyState: number;
  send(data: string): void;
  close(code?: number, reason?: string): void;
  onopen: ((event: unknown) => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: ((event: { code: number }) => void) | null;
  onerror: ((event: unknown) => void) | null;
};

export type RealtimeOptions = {
  url: string;
  getToken: () => string | null;
  /** Called after the server rejected our token (close code 4401); resolve true when a new token is ready. */
  refreshSession?: () => Promise<boolean>;
  onEvent: (event: RealtimeEvent) => void;
  /** Fired when the connection is ready again after having been lost (events are not replayed). */
  onReconnect?: () => void;
  onStatus?: (status: RealtimeStatus) => void;
  createSocket?: (url: string) => SocketLike;
  baseDelayMs?: number;
  maxDelayMs?: number;
  readyTimeoutMs?: number;
  random?: () => number;
};

export const CLOSE_UNAUTHORIZED = 4401;

const EVENT_TYPES: ReadonlySet<string> = new Set([
  "message.new",
  "message.read",
  "match.new",
  "match.removed",
  "notification.new"
]);

/** Exponential backoff with jitter: base * 2^attempt, capped, scaled into [50%, 100%]. */
export function backoffDelay(attempt: number, baseMs: number, maxMs: number, random: () => number): number {
  const raw = Math.min(maxMs, baseMs * 2 ** attempt);
  return Math.round(raw * (0.5 + random() * 0.5));
}

/** Parses a raw frame into a known event; returns null for ready/unknown/invalid frames. */
export function parseRealtimeFrame(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") return null;
  try {
    const frame = JSON.parse(raw) as { type?: unknown; data?: unknown };
    if (typeof frame.type === "string" && EVENT_TYPES.has(frame.type) && typeof frame.data === "object" && frame.data) {
      return frame as RealtimeEvent;
    }
  } catch {
    /* ignore malformed frames */
  }
  return null;
}

export class RealtimeClient {
  private socket: SocketLike | null = null;
  private attempt = 0;
  private stopped = true;
  private hadSession = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private readyTimer: ReturnType<typeof setTimeout> | null = null;
  private status: RealtimeStatus = "idle";

  constructor(private readonly options: RealtimeOptions) {}

  getStatus(): RealtimeStatus {
    return this.status;
  }

  start(): void {
    if (!this.stopped) return;
    this.stopped = false;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.clearTimers();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
      try {
        socket.close(1000);
      } catch {
        /* already closed */
      }
    }
    this.setStatus("idle");
  }

  /** Forces a fresh connection attempt now (e.g. app returned to foreground). */
  reconnectNow(): void {
    if (this.stopped || this.status === "open" || this.status === "connecting") return;
    this.clearTimers();
    this.connect();
  }

  private setStatus(status: RealtimeStatus) {
    if (this.status === status) return;
    this.status = status;
    this.options.onStatus?.(status);
  }

  private clearTimers() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.readyTimer) clearTimeout(this.readyTimer);
    this.reconnectTimer = null;
    this.readyTimer = null;
  }

  private connect() {
    const token = this.options.getToken();
    if (!token) {
      this.scheduleReconnect();
      return;
    }
    this.setStatus("connecting");
    const socket = (this.options.createSocket ?? ((url: string) => new WebSocket(url) as unknown as SocketLike))(
      this.options.url
    );
    this.socket = socket;

    socket.onopen = () => {
      // Auth must be the first frame, within 5 seconds.
      socket.send(JSON.stringify({ type: "auth", token }));
      this.readyTimer = setTimeout(() => socket.close(), this.options.readyTimeoutMs ?? 8000);
    };
    socket.onmessage = (event) => {
      if (typeof event.data === "string") {
        try {
          const frame = JSON.parse(event.data) as { type?: string };
          if (frame.type === "ready") {
            if (this.readyTimer) clearTimeout(this.readyTimer);
            this.readyTimer = null;
            this.attempt = 0;
            this.setStatus("open");
            if (this.hadSession) this.options.onReconnect?.();
            this.hadSession = true;
            return;
          }
        } catch {
          return;
        }
      }
      const parsed = parseRealtimeFrame(event.data);
      if (parsed) this.options.onEvent(parsed);
    };
    socket.onerror = () => {
      /* onclose always follows; reconnect is handled there */
    };
    socket.onclose = (event) => {
      if (this.socket !== socket) return;
      this.socket = null;
      if (this.readyTimer) clearTimeout(this.readyTimer);
      this.readyTimer = null;
      if (this.stopped) return;
      this.setStatus("closed");
      if (event.code === CLOSE_UNAUTHORIZED && this.options.refreshSession) {
        void this.options
          .refreshSession()
          .catch(() => false)
          .then(() => this.scheduleReconnect());
        return;
      }
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect() {
    if (this.stopped) return;
    const delay = backoffDelay(
      this.attempt,
      this.options.baseDelayMs ?? 1000,
      this.options.maxDelayMs ?? 30_000,
      this.options.random ?? Math.random
    );
    this.attempt += 1;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, delay);
  }
}
