import { websocketUrl } from "../api/config";
import { currentTokens } from "../api/session";
import { refreshSession } from "../api/client";

export type RealtimeEvent = { type: string; data?: unknown };
type Handler = (event: RealtimeEvent) => void;
type StatusHandler = (connected: boolean) => void;

/**
 * Resilient WebSocket client: authenticates with the first frame (the token
 * never appears in the URL), reconnects with exponential backoff and
 * refreshes the access token when the server rejects it.
 */
export class RealtimeClient {
  private ws: WebSocket | null = null;
  private attempts = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;
  private readonly handlers = new Set<Handler>();
  private readonly statusHandlers = new Set<StatusHandler>();

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
    this.ws?.close();
    this.ws = null;
    this.emitStatus(false);
  }

  /** Forces a fresh connection (e.g. app back to foreground). */
  reconnect() {
    if (this.stopped) {
      return;
    }
    this.ws?.close();
  }

  subscribe(handler: Handler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  onStatus(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    return () => this.statusHandlers.delete(handler);
  }

  private emitStatus(connected: boolean) {
    this.statusHandlers.forEach((h) => h(connected));
  }

  private connect() {
    const token = currentTokens()?.accessToken;
    if (!token || this.stopped) {
      return;
    }
    let authRejected = false;
    const ws = new WebSocket(websocketUrl());
    this.ws = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "auth", token }));
    };
    ws.onmessage = (message) => {
      let event: RealtimeEvent;
      try {
        event = JSON.parse(String(message.data)) as RealtimeEvent;
      } catch {
        return;
      }
      if (event.type === "ready") {
        this.attempts = 0;
        this.emitStatus(true);
      }
      this.handlers.forEach((h) => h(event));
    };
    ws.onclose = (event) => {
      if (this.ws === ws) {
        this.ws = null;
      }
      this.emitStatus(false);
      authRejected = event.code === 1008;
      this.schedule(authRejected);
    };
    ws.onerror = () => {
      // onclose follows and handles the retry.
    };
  }

  private schedule(authRejected: boolean) {
    if (this.stopped) {
      return;
    }
    const delay = Math.min(30_000, 1_000 * 2 ** this.attempts) + Math.random() * 500;
    this.attempts += 1;
    this.timer = setTimeout(async () => {
      this.timer = null;
      if (authRejected && !(await refreshSession())) {
        return;
      }
      this.connect();
    }, delay);
  }
}

export const realtime = new RealtimeClient();
