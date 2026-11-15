import { API_BASE_URL, freshAccessToken, refreshSession } from "./client";

export type RealtimeEvent =
  | { type: "ready"; data: { userId: string } }
  | { type: "message.new"; data: import("./types").Message }
  | { type: "conversation.read"; data: { conversationId: string; userId: string; readAt: string } }
  | { type: "match.new"; data: import("./types").MatchSummary }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.new"; data: import("./types").AppNotification };

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

/**
 * WebSocket client with in-band authentication (the token is sent as the
 * first frame, never in the URL) and exponential reconnect backoff.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private attempts = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;
  connected = false;

  start() {
    if (!this.stopped) return;
    this.stopped = false;
    void this.connect();
  }

  stop() {
    this.stopped = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.socket?.close();
    this.socket = null;
    this.setConnected(false);
  }

  /** Reconnect now (e.g. app back to foreground). */
  poke() {
    if (this.stopped || this.connected) return;
    if (this.timer) clearTimeout(this.timer);
    this.attempts = 0;
    void this.connect();
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
    if (this.connected === value) return;
    this.connected = value;
    this.statusListeners.forEach((l) => l(value));
  }

  private async connect() {
    if (this.stopped || !API_BASE_URL) return;
    const token = await freshAccessToken();
    if (!token) return;
    const url = API_BASE_URL.replace(/^http/, "ws") + "/ws";
    let socket: WebSocket;
    try {
      socket = new WebSocket(url);
    } catch {
      this.scheduleReconnect();
      return;
    }
    this.socket = socket;
    socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token }));
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
        // Access token expired or revoked: refresh, then retry.
        void refreshSession().then((ok) => (ok ? this.scheduleReconnect(0) : this.stop()));
        return;
      }
      this.scheduleReconnect();
    };
    socket.onerror = () => socket.close();
  }

  private scheduleReconnect(delay?: number) {
    if (this.stopped) return;
    const wait = delay ?? Math.min(30_000, 1000 * 2 ** this.attempts) + Math.random() * 500;
    this.attempts += 1;
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => void this.connect(), wait);
  }
}

export const realtime = new RealtimeClient();
