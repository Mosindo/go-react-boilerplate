import { parseServerFrame } from "./protocol";
import type { RealtimeEvent } from "../api";

export type RealtimeStatus = "idle" | "connecting" | "ready";

export type SocketLike = {
  onopen: (() => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: ((event: { code: number }) => void) | null;
  onerror: (() => void) | null;
  send(data: string): void;
  close(code?: number): void;
};

export type RealtimeOptions = {
  url: string;
  /** Resolves a valid access token; `forceRefresh` is set after the server rejected the previous one. */
  getToken: (forceRefresh: boolean) => Promise<string | null>;
  onEvent: (event: RealtimeEvent) => void;
  /** REST is the source of truth: called after a (re)connection that may have missed events. */
  onResync: () => void;
  onStatus?: (status: RealtimeStatus) => void;
  createSocket?: (url: string) => SocketLike;
  random?: () => number;
};

export const PING_INTERVAL_MS = 25_000;
export const AUTH_FAILURE_CLOSE_CODE = 4401;
const BASE_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

export class RealtimeService {
  private readonly options: RealtimeOptions;
  private socket: SocketLike | null = null;
  private running = false;
  private attempts = 0;
  private authFailed = false;
  private needsResync = false;
  private awaitingPong = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private generation = 0;

  constructor(options: RealtimeOptions) {
    this.options = options;
  }

  start(config: { resync?: boolean } = {}): void {
    if (this.running) {
      return;
    }
    this.running = true;
    this.needsResync = config.resync ?? false;
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.generation += 1;
    this.clearTimers();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onclose = null;
      socket.onmessage = null;
      socket.close(1000);
    }
    this.options.onStatus?.("idle");
  }

  private async connect(): Promise<void> {
    const generation = this.generation;
    this.options.onStatus?.("connecting");
    let token: string | null = null;
    try {
      token = await this.options.getToken(this.authFailed);
    } catch {
      token = null;
    }
    if (!this.running || generation !== this.generation) {
      return;
    }
    if (!token) {
      this.scheduleReconnect();
      return;
    }

    const socket = (this.options.createSocket ?? ((url) => new WebSocket(url) as unknown as SocketLike))(
      this.options.url
    );
    this.socket = socket;
    socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token }));
    socket.onmessage = (event) => this.handleFrame(event.data);
    socket.onerror = () => undefined;
    socket.onclose = (event) => {
      if (this.socket !== socket) {
        return;
      }
      this.socket = null;
      this.stopPing();
      this.needsResync = true;
      this.authFailed = event.code === AUTH_FAILURE_CLOSE_CODE;
      this.options.onStatus?.("connecting");
      if (this.running) {
        this.scheduleReconnect();
      }
    };
  }

  private handleFrame(raw: unknown): void {
    this.awaitingPong = false;
    const frame = parseServerFrame(raw);
    if (!frame) {
      return;
    }
    if (frame.kind === "ready") {
      this.attempts = 0;
      this.authFailed = false;
      this.options.onStatus?.("ready");
      this.startPing();
      if (this.needsResync) {
        this.needsResync = false;
        this.options.onResync();
      }
    } else if (frame.kind === "event") {
      this.options.onEvent(frame.event);
    }
  }

  private startPing(): void {
    this.stopPing();
    this.awaitingPong = false;
    this.pingTimer = setInterval(() => {
      const socket = this.socket;
      if (!socket) {
        return;
      }
      if (this.awaitingPong) {
        // No answer to the previous ping: the link is dead, let onclose trigger a reconnection.
        socket.close();
        return;
      }
      this.awaitingPong = true;
      socket.send(JSON.stringify({ type: "ping" }));
    }, PING_INTERVAL_MS);
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  private scheduleReconnect(): void {
    const random = this.options.random ?? Math.random;
    const ceiling = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** this.attempts);
    const delay = Math.round(ceiling * (0.5 + random() / 2));
    this.attempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (this.running) {
        void this.connect();
      }
    }, delay);
  }

  private clearTimers(): void {
    this.stopPing();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}
