import { realtimeUrl } from "./client";
import type { MatchSummary, Message, ReadReceipt, AppNotification } from "./types";

export type RealtimeEvent =
  | { type: "ready"; data: { userId: string } }
  | { type: "message.created"; data: Message }
  | { type: "conversation.read"; data: ReadReceipt }
  | { type: "match.created"; data: MatchSummary }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.created"; data: AppNotification };

type Listener = (event: RealtimeEvent) => void;

/**
 * Resilient WebSocket client: authenticates with the first frame (tokens never
 * go in URLs), reconnects with exponential backoff and asks for a token
 * refresh when the server rejects the credentials.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  constructor(
    private getToken: () => string | null,
    private refreshToken: () => Promise<string | null>
  ) {}

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  start() {
    if (!this.stopped) {
      return;
    }
    this.stopped = false;
    this.connect();
  }

  stop() {
    this.stopped = true;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
  }

  /** Reconnect now (e.g. app returned to foreground). */
  poke() {
    if (!this.stopped && (!this.socket || this.socket.readyState === WebSocket.CLOSED)) {
      this.retry = 0;
      this.connect();
    }
  }

  private connect() {
    const token = this.getToken();
    if (!token || this.stopped) {
      return;
    }
    let authRejected = false;
    const socket = new WebSocket(realtimeUrl());
    this.socket = socket;

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (message) => {
      try {
        const event = JSON.parse(String(message.data)) as RealtimeEvent;
        if (event.type === "ready") {
          this.retry = 0;
        }
        this.listeners.forEach((listener) => listener(event));
      } catch {
        // Ignore malformed frames.
      }
    };
    socket.onclose = (event) => {
      if (this.socket === socket) {
        this.socket = null;
      }
      authRejected = event.code === 1008;
      this.scheduleReconnect(authRejected);
    };
    socket.onerror = () => {
      socket.close();
    };
  }

  private scheduleReconnect(refresh: boolean) {
    if (this.stopped || this.timer) {
      return;
    }
    const delay = Math.min(30_000, 1000 * 2 ** this.retry) + Math.random() * 500;
    this.retry += 1;
    this.timer = setTimeout(async () => {
      this.timer = null;
      if (refresh) {
        await this.refreshToken();
      }
      this.connect();
    }, delay);
  }
}
