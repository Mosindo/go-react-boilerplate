import { API_BASE_URL, refreshAccessToken } from "./client";
import { getTokens } from "../store/tokenStore";

export type RealtimeEvent = {
  type: string;
  at: string;
  payload?: unknown;
};

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

const UNAUTHORIZED_CLOSE = 4401;

/**
 * WebSocket client: authenticates with the first frame (the token never
 * appears in the URL), reconnects with exponential backoff and refreshes
 * the access token when the server rejects it.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;
  connected = false;

  start() {
    if (!this.stopped) {
      return;
    }
    this.stopped = false;
    this.open();
  }

  stop() {
    this.stopped = true;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
    this.setConnected(false);
  }

  /** Forces a fresh connection (e.g. when the app returns to foreground). */
  reconnect() {
    if (this.stopped) {
      return;
    }
    this.retry = 0;
    this.socket?.close();
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  onStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    return () => this.statusListeners.delete(listener);
  }

  private setConnected(value: boolean) {
    if (this.connected !== value) {
      this.connected = value;
      this.statusListeners.forEach((l) => l(value));
    }
  }

  private open() {
    const token = getTokens()?.accessToken;
    if (this.stopped || !token || !API_BASE_URL) {
      return;
    }
    const url = `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
    let socket: WebSocket;
    try {
      socket = new WebSocket(url);
    } catch {
      this.schedule();
      return;
    }
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
        this.retry = 0;
        this.setConnected(true);
      }
      this.listeners.forEach((l) => l(event));
    };
    socket.onclose = (event) => {
      if (this.socket === socket) {
        this.socket = null;
      }
      this.setConnected(false);
      if (this.stopped) {
        return;
      }
      if (event.code === UNAUTHORIZED_CLOSE) {
        void refreshAccessToken().then((ok) => {
          if (ok) {
            this.schedule(0);
          }
        });
        return;
      }
      this.schedule();
    };
    socket.onerror = () => {
      socket.close();
    };
  }

  private schedule(delay?: number) {
    if (this.stopped) {
      return;
    }
    const wait = delay ?? Math.min(30_000, 1000 * 2 ** this.retry) + Math.floor(Math.random() * 500);
    this.retry = Math.min(this.retry + 1, 6);
    if (this.timer) {
      clearTimeout(this.timer);
    }
    this.timer = setTimeout(() => {
      this.timer = null;
      this.open();
    }, wait);
  }
}

export const realtime = new RealtimeClient();
