import { backoffDelay } from "../utils/backoff";
import { websocketUrl } from "../utils/url";
import { API_BASE_URL } from "./client";
import { tokenStore } from "./tokenStore";
import type { RealtimeEvent } from "./types";

type Listener = (event: RealtimeEvent) => void;

const PING_EVERY_MS = 25_000;

function parseEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") return null;
  try {
    const value: unknown = JSON.parse(raw);
    if (typeof value === "object" && value !== null && typeof (value as { type?: unknown }).type === "string") {
      return value as RealtimeEvent;
    }
  } catch {
    // ignore malformed frames
  }
  return null;
}

/**
 * Authenticated WebSocket with automatic reconnection (exponential backoff + jitter).
 * The access token is sent in the first frame, never in the URL.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private attempt = 0;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private running = false;
  private refreshAccessToken: () => Promise<string | null>;

  constructor(refreshAccessToken: () => Promise<string | null>) {
    this.refreshAccessToken = refreshAccessToken;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  start(): void {
    if (this.running || !API_BASE_URL) return;
    this.running = true;
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.clearTimers();
    this.socket?.close();
    this.socket = null;
  }

  /** Forces an immediate reconnection (e.g. app returned to foreground). */
  reconnectNow(): void {
    if (!this.running) return;
    this.clearTimers();
    this.socket?.close();
    this.socket = null;
    this.attempt = 0;
    void this.connect();
  }

  private clearTimers(): void {
    if (this.pingTimer) clearInterval(this.pingTimer);
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.pingTimer = null;
    this.retryTimer = null;
  }

  private scheduleRetry(): void {
    if (!this.running || this.retryTimer) return;
    const delay = backoffDelay(this.attempt);
    this.attempt += 1;
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      void this.connect();
    }, delay);
  }

  private async connect(): Promise<void> {
    if (!this.running) return;
    const token = (await this.refreshAccessToken()) ?? (await tokenStore.get())?.accessToken ?? null;
    if (!token || !this.running) {
      this.scheduleRetry();
      return;
    }
    const ws = new WebSocket(websocketUrl(API_BASE_URL));
    this.socket = ws;
    ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
    ws.onmessage = (msg: { data: unknown }) => {
      const event = parseEvent(msg.data);
      if (!event) return;
      if (event.type === "ready") {
        this.attempt = 0;
        this.pingTimer = setInterval(() => {
          if (ws.readyState === 1) ws.send(JSON.stringify({ type: "ping" }));
        }, PING_EVERY_MS);
      }
      this.listeners.forEach((l) => l(event));
    };
    ws.onclose = () => {
      if (this.socket !== ws) return;
      this.socket = null;
      if (this.pingTimer) clearInterval(this.pingTimer);
      this.pingTimer = null;
      this.scheduleRetry();
    };
    ws.onerror = () => ws.close();
  }
}
