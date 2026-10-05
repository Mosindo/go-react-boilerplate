import type { RealtimeEvent } from "../api/types";

export type RealtimeStatus = "idle" | "connecting" | "connected" | "reconnecting";

/** The subset of the WebSocket API the client uses (lets tests inject a fake). */
export interface SocketLike {
  onopen: (() => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: ((event: { code: number }) => void) | null;
  onerror: (() => void) | null;
  send(data: string): void;
  close(): void;
}

export type RealtimeDeps = {
  url: string;
  getAccessToken: () => Promise<string | null>;
  /** Refreshes the session and returns the new access token, or null when the session is over. */
  refreshAccessToken: () => Promise<string | null>;
  createSocket: (url: string) => SocketLike;
  onEvent: (event: RealtimeEvent) => void;
  onStatus?: (status: RealtimeStatus) => void;
  /** Called every time the socket becomes ready; `isReconnect` tells the app to catch up on missed events. */
  onConnected?: (isReconnect: boolean) => void;
  random?: () => number;
  setTimer?: (fn: () => void, ms: number) => unknown;
  clearTimer?: (handle: unknown) => void;
};

const AUTH_REJECTED = 4401;
const BASE_DELAY_MS = 1_000;
const MAX_DELAY_MS = 30_000;

/**
 * Keeps one authenticated WebSocket open: authenticates with the first frame, reconnects with exponential
 * backoff and jitter, refreshes the access token when the server rejects it (they expire every 15 minutes),
 * and reports reconnects so the app can refetch what it missed while offline.
 */
export class RealtimeClient {
  private socket: SocketLike | null = null;
  private status: RealtimeStatus = "idle";
  private attempts = 0;
  private timer: unknown = null;
  private running = false;
  private everConnected = false;
  private generation = 0;
  private authRetries = 0;

  constructor(private readonly deps: RealtimeDeps) {}

  get currentStatus(): RealtimeStatus {
    return this.status;
  }

  start(): void {
    if (this.running) return;
    this.running = true;
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.generation++;
    this.clearTimer();
    this.detach();
    this.setStatus("idle");
  }

  /** Reconnects right away if the socket is down (app returned to the foreground, network came back). */
  reconnectNow(): void {
    if (!this.running || this.status === "connected" || this.status === "connecting") return;
    this.clearTimer();
    void this.connect();
  }

  private setStatus(status: RealtimeStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.deps.onStatus?.(status);
  }

  private async connect(token?: string | null): Promise<void> {
    const generation = ++this.generation;
    this.setStatus(this.everConnected ? "reconnecting" : "connecting");
    const accessToken = token ?? (await this.deps.getAccessToken());
    if (!this.running || generation !== this.generation) return;
    if (!accessToken) {
      this.running = false;
      this.setStatus("idle");
      return;
    }

    this.detach();
    const socket = this.deps.createSocket(this.deps.url);
    this.socket = socket;
    socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token: accessToken }));
    socket.onmessage = (message) => this.handleMessage(generation, message.data);
    socket.onclose = (event) => void this.handleClose(generation, event.code);
    socket.onerror = () => {
      // A close event always follows an error; reconnection is handled there.
    };
  }

  private handleMessage(generation: number, raw: unknown): void {
    if (generation !== this.generation || typeof raw !== "string") return;
    let event: RealtimeEvent;
    try {
      event = JSON.parse(raw) as RealtimeEvent;
    } catch {
      return;
    }
    if (event.type === "ready") {
      const isReconnect = this.everConnected;
      this.everConnected = true;
      this.attempts = 0;
      this.authRetries = 0;
      this.setStatus("connected");
      this.deps.onConnected?.(isReconnect);
      return;
    }
    this.deps.onEvent(event);
  }

  private async handleClose(generation: number, code: number): Promise<void> {
    if (generation !== this.generation || !this.running) return;
    this.socket = null;
    this.setStatus(this.everConnected ? "reconnecting" : "connecting");

    if (code === AUTH_REJECTED && this.authRetries < 2) {
      this.authRetries++;
      const fresh = await this.deps.refreshAccessToken().catch(() => null);
      if (!this.running || generation !== this.generation) return;
      if (fresh) {
        void this.connect(fresh);
        return;
      }
      // The session is gone for good (the auth layer signs the user out); fall through to a slow retry in
      // case the failure was only a network problem.
    }
    this.scheduleReconnect();
  }

  private scheduleReconnect(): void {
    this.clearTimer();
    const random = this.deps.random ?? Math.random;
    const exponential = Math.min(MAX_DELAY_MS, BASE_DELAY_MS * 2 ** this.attempts);
    const delay = Math.round(exponential / 2 + (random() * exponential) / 2); // 50–100 % of the step
    this.attempts++;
    const setTimer = this.deps.setTimer ?? ((fn, ms) => setTimeout(fn, ms));
    this.timer = setTimer(() => {
      this.timer = null;
      void this.connect();
    }, delay);
  }

  private clearTimer(): void {
    if (this.timer !== null) {
      (this.deps.clearTimer ?? ((handle) => clearTimeout(handle as ReturnType<typeof setTimeout>)))(this.timer);
      this.timer = null;
    }
  }

  private detach(): void {
    const socket = this.socket;
    this.socket = null;
    if (!socket) return;
    socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null;
    try {
      socket.close();
    } catch {
      // Already closed.
    }
  }
}
