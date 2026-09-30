import { AppState, type AppStateStatus } from "react-native";
import { websocketUrl } from "./config";
import { realtimeApi } from "./endpoints";

export type RealtimeEvent = {
  type: string;
  conversationId?: string;
  data?: unknown;
  truncated?: boolean;
};

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

/**
 * WebSocket client with ticket authentication, exponential backoff and
 * foreground/background awareness. Consumers fall back to polling while
 * `connected` is false.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private running = false;
  private appStateSub: { remove: () => void } | null = null;
  connected = false;

  start(): void {
    if (this.running) {
      return;
    }
    this.running = true;
    this.appStateSub = AppState.addEventListener("change", this.onAppState);
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.appStateSub?.remove();
    this.appStateSub = null;
    if (this.timer) {
      clearTimeout(this.timer);
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

  private onAppState = (state: AppStateStatus) => {
    if (state === "active" && this.running && !this.socket) {
      this.retry = 0;
      void this.connect();
    }
  };

  private setConnected(value: boolean) {
    if (this.connected !== value) {
      this.connected = value;
      this.statusListeners.forEach((listener) => listener(value));
    }
  }

  private async connect(): Promise<void> {
    if (!this.running || this.socket) {
      return;
    }
    try {
      const ticket = await realtimeApi.ticket();
      if (!this.running) {
        return;
      }
      const socket = new WebSocket(websocketUrl(`/realtime?ticket=${encodeURIComponent(ticket)}`));
      this.socket = socket;
      socket.onopen = () => {
        this.retry = 0;
        this.setConnected(true);
      };
      socket.onmessage = (message) => {
        try {
          const event = JSON.parse(String(message.data)) as RealtimeEvent;
          this.listeners.forEach((listener) => listener(event));
        } catch {
          // Ignore malformed frames.
        }
      };
      socket.onclose = () => {
        this.socket = null;
        this.setConnected(false);
        this.scheduleReconnect();
      };
      socket.onerror = () => socket.close();
    } catch {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (!this.running || AppState.currentState === "background") {
      return;
    }
    const delay = Math.min(30_000, 1000 * 2 ** this.retry) + Math.random() * 500;
    this.retry += 1;
    this.timer = setTimeout(() => {
      void this.connect();
    }, delay);
  }
}
