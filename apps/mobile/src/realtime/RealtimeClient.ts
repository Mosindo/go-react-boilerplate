import { API_BASE_URL, refreshSession } from "../api/client";
import { getTokens } from "../api/session";

export type RealtimeEvent = { type: string; data: unknown; at: string };
type Listener = (event: RealtimeEvent) => void;

const MAX_BACKOFF_MS = 30_000;

/**
 * WebSocket client with automatic reconnection (exponential backoff). The access token is
 * sent as the first frame, never in the URL. When the server closes with 4001 (token
 * expired/invalid) the session is refreshed before reconnecting.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<(connected: boolean) => void>();
  private attempts = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  start() {
    if (!this.stopped) return;
    this.stopped = false;
    this.connect();
  }

  stop() {
    this.stopped = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.socket?.close();
    this.socket = null;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  onStatus(listener: (connected: boolean) => void): () => void {
    this.statusListeners.add(listener);
    return () => this.statusListeners.delete(listener);
  }

  private setConnected(connected: boolean) {
    this.statusListeners.forEach((l) => l(connected));
  }

  private connect() {
    const token = getTokens()?.accessToken;
    if (this.stopped || !token || !API_BASE_URL) return;
    const url = `${API_BASE_URL.replace(/^http/, "ws")}/realtime`;
    const socket = new WebSocket(url);
    this.socket = socket;

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (message) => {
      let event: RealtimeEvent;
      try {
        event = JSON.parse(String(message.data)) as RealtimeEvent;
      } catch {
        return;
      }
      if (event.type === "ready") {
        this.attempts = 0;
        this.setConnected(true);
      }
      this.listeners.forEach((l) => l(event));
    };
    socket.onclose = (event) => {
      if (this.socket === socket) this.socket = null;
      this.setConnected(false);
      if (this.stopped) return;
      if (event.code === 4001) {
        void refreshSession().then((ok) => {
          if (ok) this.scheduleReconnect(0);
        });
        return;
      }
      this.scheduleReconnect();
    };
    socket.onerror = () => {
      socket.close();
    };
  }

  private scheduleReconnect(delay?: number) {
    if (this.timer) clearTimeout(this.timer);
    const backoff = delay ?? Math.min(MAX_BACKOFF_MS, 1000 * 2 ** this.attempts) * (0.75 + Math.random() * 0.5);
    this.attempts += 1;
    this.timer = setTimeout(() => this.connect(), backoff);
  }
}
