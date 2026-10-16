import { API_BASE_URL } from "./client";
import { endpoints } from "./endpoints";
import { requestRealtimeTicket } from "./platform";
import type { RealtimeEvent } from "./types";

export type RealtimeStatus = "connecting" | "open" | "closed";

type Handlers = {
  onEvent: (event: RealtimeEvent) => void;
  onStatus: (status: RealtimeStatus) => void;
};

const RECONNECT_BASE_MS = 1_000;
const RECONNECT_MAX_MS = 30_000;

export function toSocketUrl(apiBaseUrl: string, ticket: string): string {
  const base = apiBaseUrl.replace(/^http/, "ws");
  return `${base}${endpoints.realtime.socket}?ticket=${encodeURIComponent(ticket)}`;
}

export function reconnectDelay(attempt: number): number {
  return Math.min(RECONNECT_MAX_MS, RECONNECT_BASE_MS * 2 ** attempt);
}

export function parseEvent(raw: string): RealtimeEvent | null {
  try {
    const parsed = JSON.parse(raw) as { type?: string; data?: unknown };
    switch (parsed.type) {
      case "message.new":
      case "message.read":
      case "match.new":
      case "notification.new":
        return parsed as RealtimeEvent;
      default:
        return null;
    }
  } catch {
    return null;
  }
}

/**
 * One WebSocket per signed-in session. Each (re)connect asks the API for a fresh
 * single-use ticket, so the access token never appears in a URL.
 */
export class RealtimeConnection {
  private socket: WebSocket | null = null;
  private attempt = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  constructor(private readonly handlers: Handlers) {}

  start(): void {
    if (!this.stopped) {
      return;
    }
    this.stopped = false;
    void this.connect();
  }

  stop(): void {
    this.stopped = true;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
    this.handlers.onStatus("closed");
  }

  /** Called when the app returns to the foreground: sockets die silently in the background. */
  reconnectNow(): void {
    if (this.stopped) {
      return;
    }
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
    void this.connect();
  }

  private async connect(): Promise<void> {
    if (this.stopped) {
      return;
    }
    this.handlers.onStatus("connecting");
    try {
      const { ticket } = await requestRealtimeTicket();
      if (this.stopped) {
        return;
      }
      const socket = new WebSocket(toSocketUrl(API_BASE_URL, ticket));
      this.socket = socket;
      socket.onopen = () => {
        this.attempt = 0;
        this.handlers.onStatus("open");
      };
      socket.onmessage = (message: WebSocketMessageEvent) => {
        if (typeof message.data !== "string") {
          return;
        }
        const event = parseEvent(message.data);
        if (event) {
          this.handlers.onEvent(event);
        }
      };
      socket.onclose = () => {
        if (this.socket === socket) {
          this.socket = null;
          this.handlers.onStatus("closed");
          this.scheduleReconnect();
        }
      };
      socket.onerror = () => {
        socket.close();
      };
    } catch {
      this.handlers.onStatus("closed");
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.timer) {
      return;
    }
    const delay = reconnectDelay(this.attempt);
    this.attempt += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      void this.connect();
    }, delay);
  }
}
