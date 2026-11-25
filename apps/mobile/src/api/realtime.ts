import { api } from "./client";
import { API_BASE_URL } from "./config";

export type RealtimeEvent =
  | { type: "ready"; data: Record<string, never> }
  | { type: "message.created"; data: import("./types").Message }
  | { type: "conversation.read"; data: { conversationId: string; userId: string; readAt: string } }
  | { type: "match.created"; data: { matchId: string; conversationId: string; userId: string } }
  | { type: "match.ended"; data: { matchId: string } }
  | { type: "notification.created"; data: import("./types").AppNotification };

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

/**
 * Maintains one WebSocket to the API. Authentication uses a single-use ticket
 * fetched with the bearer token; the socket reconnects with exponential
 * backoff while the client is started.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private running = false;
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  connected = false;

  start() {
    if (this.running) {
      return;
    }
    this.running = true;
    void this.connect();
  }

  stop() {
    this.running = false;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
    this.setConnected(false);
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  onStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  private setConnected(value: boolean) {
    if (this.connected !== value) {
      this.connected = value;
      this.statusListeners.forEach((l) => l(value));
    }
  }

  private scheduleReconnect() {
    if (!this.running) {
      return;
    }
    const delay = Math.min(30_000, 1_000 * 2 ** this.retry);
    this.retry += 1;
    this.timer = setTimeout(() => void this.connect(), delay);
  }

  private async connect() {
    if (!this.running) {
      return;
    }
    try {
      const { ticket } = await api<{ ticket: string }>("/realtime/ticket", { method: "POST" });
      if (!this.running) {
        return;
      }
      const wsUrl = `${API_BASE_URL.replace(/^http/, "ws")}/realtime?ticket=${encodeURIComponent(ticket)}`;
      const socket = new WebSocket(wsUrl);
      this.socket = socket;
      socket.onopen = () => {
        this.retry = 0;
        this.setConnected(true);
      };
      socket.onmessage = (message) => {
        try {
          const event = JSON.parse(String(message.data)) as RealtimeEvent;
          this.listeners.forEach((l) => l(event));
        } catch {
          // Ignore malformed frames.
        }
      };
      socket.onclose = () => {
        if (this.socket === socket) {
          this.socket = null;
        }
        this.setConnected(false);
        this.scheduleReconnect();
      };
      socket.onerror = () => {
        socket.close();
      };
    } catch {
      this.setConnected(false);
      this.scheduleReconnect();
    }
  }
}

export const realtime = new RealtimeClient();
