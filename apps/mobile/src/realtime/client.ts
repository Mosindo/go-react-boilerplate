import type { RealtimeEvent } from '../api/types';

export type SocketLike = {
  onopen: ((ev: unknown) => void) | null;
  onmessage: ((ev: { data: unknown }) => void) | null;
  onclose: ((ev: unknown) => void) | null;
  onerror: ((ev: unknown) => void) | null;
  close(): void;
};

export type RealtimeStatus = 'idle' | 'connecting' | 'open' | 'reconnecting';

export type RealtimeClientOptions = {
  getTicket: () => Promise<string>;
  /** ws(s)://host — without path. */
  wsBaseUrl: string;
  onEvent: (event: RealtimeEvent) => void;
  /** Called on every successful open. `isReconnect` is false for the first one. */
  onOpen?: (isReconnect: boolean) => void;
  onStatus?: (status: RealtimeStatus) => void;
  createSocket?: (url: string) => SocketLike;
  baseDelayMs?: number;
  maxDelayMs?: number;
  random?: () => number;
};

const KNOWN: ReadonlySet<string> = new Set([
  'message.new',
  'message.read',
  'match.new',
  'match.removed',
  'notification.new',
]);

export function parseFrame(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== 'string') return null;
  try {
    const frame = JSON.parse(raw) as { type?: unknown; data?: unknown };
    if (typeof frame.type !== 'string' || !KNOWN.has(frame.type)) return null;
    if (typeof frame.data !== 'object' || frame.data === null) return null;
    return frame as RealtimeEvent;
  } catch {
    return null;
  }
}

export function backoffDelay(
  attempt: number,
  baseMs: number,
  maxMs: number,
  random: () => number = Math.random,
): number {
  const exp = Math.min(maxMs, baseMs * 2 ** attempt);
  return Math.round(exp * (0.5 + random() * 0.5));
}

export class RealtimeClient {
  private socket: SocketLike | null = null;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private attempt = 0;
  private generation = 0;
  private running = false;
  private everOpened = false;
  private status: RealtimeStatus = 'idle';

  constructor(private readonly opts: RealtimeClientOptions) {}

  start(): void {
    if (this.running) return;
    this.running = true;
    void this.connect();
  }

  stop(): void {
    this.running = false;
    this.generation += 1;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    this.detach();
    this.setStatus('idle');
  }

  isOpen(): boolean {
    return this.status === 'open';
  }

  /** Reconnect right away (app returned to foreground) unless already connected. */
  reconnectNow(): void {
    if (!this.running || this.status === 'open' || this.status === 'connecting') return;
    if (this.timer) clearTimeout(this.timer);
    this.timer = null;
    void this.connect();
  }

  private setStatus(status: RealtimeStatus) {
    if (this.status === status) return;
    this.status = status;
    this.opts.onStatus?.(status);
  }

  private detach() {
    const s = this.socket;
    this.socket = null;
    if (!s) return;
    s.onopen = s.onmessage = s.onclose = s.onerror = null;
    try {
      s.close();
    } catch {
      // ignore
    }
  }

  private async connect(): Promise<void> {
    const gen = ++this.generation;
    this.setStatus(this.everOpened ? 'reconnecting' : 'connecting');
    let ticket: string;
    try {
      ticket = await this.opts.getTicket();
    } catch {
      if (gen === this.generation && this.running) this.scheduleReconnect();
      return;
    }
    if (gen !== this.generation || !this.running) return;
    const url = `${this.opts.wsBaseUrl}/ws?ticket=${encodeURIComponent(ticket)}`;
    const socket = this.opts.createSocket
      ? this.opts.createSocket(url)
      : (new WebSocket(url) as unknown as SocketLike);
    this.socket = socket;
    socket.onopen = () => {
      if (gen !== this.generation) return;
      this.attempt = 0;
      const isReconnect = this.everOpened;
      this.everOpened = true;
      this.setStatus('open');
      this.opts.onOpen?.(isReconnect);
    };
    socket.onmessage = (ev) => {
      if (gen !== this.generation) return;
      const event = parseFrame(ev.data);
      if (event) this.opts.onEvent(event);
    };
    const onDown = () => {
      if (gen !== this.generation) return;
      this.detach();
      if (this.running) this.scheduleReconnect();
    };
    socket.onclose = onDown;
    socket.onerror = onDown;
  }

  private scheduleReconnect() {
    this.setStatus('reconnecting');
    if (this.timer) clearTimeout(this.timer);
    const delay = backoffDelay(
      this.attempt,
      this.opts.baseDelayMs ?? 1000,
      this.opts.maxDelayMs ?? 30_000,
      this.opts.random,
    );
    this.attempt += 1;
    this.timer = setTimeout(() => {
      this.timer = null;
      if (this.running) void this.connect();
    }, delay);
  }
}
