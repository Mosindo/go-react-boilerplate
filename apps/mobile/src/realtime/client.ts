import type { RealtimeEvent } from '../api/types';
import { backoffDelay } from './backoff';
import { parseFrame } from './parse';

export const PING_INTERVAL_MS = 25_000;

export type RealtimeStatus = 'idle' | 'connecting' | 'open' | 'reconnecting';

export interface SocketHandle {
  send(data: string): void;
  close(): void;
}

export interface SocketHandlers {
  onOpen(): void;
  onMessage(data: unknown): void;
  onClose(): void;
}

export interface RealtimeDeps {
  /** Fetches a single-use WS ticket and returns the full ws:// URL. */
  getUrl(): Promise<string>;
  openSocket(url: string, handlers: SocketHandlers): SocketHandle;
  onEvent(event: RealtimeEvent): void;
  /** Fired when the socket opens again after a drop: caller refetches. */
  onReconnected(): void;
  onStatus?(status: RealtimeStatus): void;
  random?: () => number;
}

/**
 * WebSocket connection manager: ticket -> socket, ping every 25 s, exponential backoff reconnect.
 */
export class RealtimeClient {
  private socket: SocketHandle | null = null;
  private generation = 0;
  private attempt = 0;
  private stopped = true;
  private hasOpened = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private status: RealtimeStatus = 'idle';

  constructor(private readonly deps: RealtimeDeps) {}

  getStatus(): RealtimeStatus {
    return this.status;
  }

  start(): void {
    if (!this.stopped) return;
    this.stopped = false;
    this.hasOpened = false;
    this.attempt = 0;
    void this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.generation += 1;
    this.clearTimers();
    this.socket?.close();
    this.socket = null;
    this.setStatus('idle');
  }

  /** Reconnect immediately when the app returns to the foreground and the socket is down. */
  ensureConnected(): void {
    if (this.stopped || this.status === 'open' || this.status === 'connecting') return;
    this.clearTimers();
    this.attempt = 0;
    void this.connect();
  }

  private setStatus(status: RealtimeStatus): void {
    if (this.status === status) return;
    this.status = status;
    this.deps.onStatus?.(status);
  }

  private clearTimers(): void {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.pingTimer) clearInterval(this.pingTimer);
    this.reconnectTimer = null;
    this.pingTimer = null;
  }

  private scheduleReconnect(): void {
    if (this.stopped) return;
    this.setStatus('reconnecting');
    const delay = backoffDelay(this.attempt, this.deps.random);
    this.attempt += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect();
    }, delay);
  }

  private async connect(): Promise<void> {
    if (this.stopped) return;
    const generation = ++this.generation;
    this.setStatus(this.hasOpened ? 'reconnecting' : 'connecting');

    let url: string;
    try {
      url = await this.deps.getUrl();
    } catch {
      if (generation === this.generation) this.scheduleReconnect();
      return;
    }
    if (this.stopped || generation !== this.generation) return;

    this.socket = this.deps.openSocket(url, {
      onOpen: () => {
        if (generation !== this.generation) return;
        this.attempt = 0;
        const reconnected = this.hasOpened;
        this.hasOpened = true;
        this.setStatus('open');
        this.pingTimer = setInterval(() => {
          try {
            this.socket?.send(JSON.stringify({ type: 'ping' }));
          } catch {
            // A failed ping surfaces as a close event.
          }
        }, PING_INTERVAL_MS);
        if (reconnected) this.deps.onReconnected();
      },
      onMessage: (data) => {
        if (generation !== this.generation) return;
        const event = parseFrame(data);
        if (event) this.deps.onEvent(event);
      },
      onClose: () => {
        if (generation !== this.generation) return;
        this.clearTimers();
        this.socket = null;
        this.scheduleReconnect();
      },
    });
  }
}
