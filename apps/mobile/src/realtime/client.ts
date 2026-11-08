import { websocketUrl } from "../config";
import { currentTokens } from "../auth/tokenStore";
import { refreshTokens } from "../api/client";
import type { RealtimeEvent } from "../api/types";

type Listener = (event: RealtimeEvent) => void;
type StatusListener = (connected: boolean) => void;

/**
 * WebSocket connection with authentication frame, exponential backoff reconnection and
 * a resync callback (the app refetches REST data after every reconnect, so no event is lost).
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private statusListeners = new Set<StatusListener>();
  private attempt = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  on(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  onStatus(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    return () => this.statusListeners.delete(listener);
  }

  start(): void {
    if (!this.stopped) return;
    this.stopped = false;
    this.connect();
  }

  stop(): void {
    this.stopped = true;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.socket?.close();
    this.socket = null;
  }

  private emitStatus(connected: boolean): void {
    this.statusListeners.forEach((l) => l(connected));
  }

  private connect(): void {
    const tokens = currentTokens();
    if (this.stopped || !tokens) return;
    let socket: WebSocket;
    try {
      socket = new WebSocket(websocketUrl());
    } catch {
      this.scheduleReconnect();
      return;
    }
    this.socket = socket;
    let authenticated = false;

    socket.onopen = () => {
      socket.send(
        JSON.stringify({ type: "auth", token: currentTokens()?.accessToken ?? tokens.accessToken })
      );
    };
    socket.onmessage = (msg: MessageEvent) => {
      let event: RealtimeEvent;
      try {
        event = JSON.parse(String(msg.data)) as RealtimeEvent;
      } catch {
        return;
      }
      if (event.type === "ready") {
        authenticated = true;
        this.attempt = 0;
        this.emitStatus(true);
      }
      this.listeners.forEach((l) => l(event));
    };
    socket.onclose = async () => {
      if (this.socket === socket) this.socket = null;
      this.emitStatus(false);
      if (this.stopped) return;
      // Closed before "ready": most likely an expired access token — refresh before retrying.
      if (!authenticated) await refreshTokens();
      this.scheduleReconnect();
    };
    socket.onerror = () => socket.close();
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.timer) return;
    const delay = Math.min(30_000, 1000 * 2 ** this.attempt) + Math.random() * 500;
    this.attempt += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      this.connect();
    }, delay);
  }
}
