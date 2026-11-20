import { API_BASE_URL } from "./client";
import { authApi } from "./endpoints";
import type { RealtimeEvent } from "./types";

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

/**
 * Keeps one WebSocket open while the user is signed in. Each connection uses
 * a fresh 60-second ticket, so no long-lived token ever appears in a URL.
 * Reconnects with exponential backoff; REST polling covers the gaps.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private retry = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private running = false;
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
    }
    this.timer = null;
    this.socket?.close();
    this.socket = null;
    this.setConnected(false);
  }

  subscribe(listener: Listener) {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  onStatus(listener: StatusListener) {
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
    const delay = Math.min(30_000, 1000 * 2 ** this.retry) + Math.random() * 500;
    this.retry += 1;
    this.timer = setTimeout(() => void this.connect(), delay);
  }

  private async connect() {
    if (!this.running || !API_BASE_URL) {
      return;
    }
    let ticket: string;
    try {
      ticket = (await authApi.realtimeTicket()).ticket;
    } catch {
      this.scheduleReconnect();
      return;
    }
    if (!this.running) {
      return;
    }
    const url = `${API_BASE_URL.replace(/^http/, "ws")}/realtime?ticket=${encodeURIComponent(ticket)}`;
    const socket = new WebSocket(url);
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
        this.setConnected(false);
        this.scheduleReconnect();
      }
    };
    socket.onerror = () => socket.close();
  }
}

export const realtime = new RealtimeClient();
