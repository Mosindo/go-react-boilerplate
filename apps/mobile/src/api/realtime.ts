import type { AppNotification, MatchSummary, Message, RealtimeEvent } from "./types";

export const PING_INTERVAL_MS = 25_000;
const PONG_TIMEOUT_MS = 10_000;
const BASE_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function hasStrings(value: Record<string, unknown>, keys: string[]): boolean {
  return keys.every((key) => typeof value[key] === "string");
}

function isMessage(value: unknown): value is Message {
  return (
    isRecord(value) &&
    hasStrings(value, ["id", "conversationId", "senderId", "body", "createdAt"]) &&
    (value.readAt === null || typeof value.readAt === "string")
  );
}

function isMatchSummary(value: unknown): value is MatchSummary {
  return (
    isRecord(value) &&
    hasStrings(value, ["matchId", "conversationId", "createdAt"]) &&
    isRecord(value.user) &&
    typeof value.user.userId === "string" &&
    typeof value.unreadCount === "number"
  );
}

function isNotification(value: unknown): value is AppNotification {
  return (
    isRecord(value) &&
    hasStrings(value, ["id", "type", "title", "body", "createdAt"]) &&
    isRecord(value.data) &&
    (value.readAt === null || typeof value.readAt === "string")
  );
}

export type ServerFrame = { kind: "ready" } | { kind: "pong" } | { kind: "event"; event: RealtimeEvent };

/** Parses and validates one raw WebSocket frame. Returns null for anything unexpected. */
export function parseServerFrame(raw: string): ServerFrame | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(parsed) || typeof parsed.type !== "string") {
    return null;
  }
  const data = parsed.data;
  switch (parsed.type) {
    case "ready":
      return { kind: "ready" };
    case "pong":
      return { kind: "pong" };
    case "message.new":
      return isMessage(data) ? { kind: "event", event: { type: "message.new", data } } : null;
    case "messages.read":
      return isRecord(data) && hasStrings(data, ["conversationId", "readerId"])
        ? {
            kind: "event",
            event: {
              type: "messages.read",
              data: { conversationId: String(data.conversationId), readerId: String(data.readerId) }
            }
          }
        : null;
    case "match.new":
      return isMatchSummary(data) ? { kind: "event", event: { type: "match.new", data } } : null;
    case "match.removed":
      return isRecord(data) && hasStrings(data, ["matchId", "conversationId"])
        ? {
            kind: "event",
            event: {
              type: "match.removed",
              data: { matchId: String(data.matchId), conversationId: String(data.conversationId) }
            }
          }
        : null;
    case "notification.new":
      return isNotification(data) ? { kind: "event", event: { type: "notification.new", data } } : null;
    default:
      return null;
  }
}

export function backoffDelayMs(attempt: number, random: () => number = Math.random): number {
  const exponential = Math.min(MAX_BACKOFF_MS, BASE_BACKOFF_MS * 2 ** attempt);
  return Math.round(exponential * (0.75 + random() * 0.25));
}

export type SocketLike = {
  readyState: number;
  onopen: (() => void) | null;
  onmessage: ((event: { data: unknown }) => void) | null;
  onclose: (() => void) | null;
  onerror: (() => void) | null;
  send: (data: string) => void;
  close: () => void;
};

export type RealtimeOptions = {
  url: string;
  getToken: () => Promise<string | null>;
  onEvent: (event: RealtimeEvent) => void;
  /** Called after every successful reconnect so callers can refetch what they missed. */
  onReconnected: () => void;
  createSocket?: (url: string) => SocketLike;
};

const OPEN = 1;

export class RealtimeClient {
  private readonly options: RealtimeOptions;
  private socket: SocketLike | null = null;
  private active = false;
  private attempts = 0;
  private hasBeenReady = false;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(options: RealtimeOptions) {
    this.options = options;
  }

  start(): void {
    if (this.active) {
      return;
    }
    this.active = true;
    void this.connect();
  }

  stop(): void {
    this.active = false;
    this.attempts = 0;
    this.clearTimers();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onclose = null;
      socket.onerror = null;
      socket.close();
    }
  }

  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
    if (this.pongTimer) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
  }

  private async connect(): Promise<void> {
    let token: string | null = null;
    try {
      token = await this.options.getToken();
    } catch {
      token = null;
    }
    if (!this.active) {
      return;
    }
    if (!token) {
      this.scheduleReconnect();
      return;
    }
    const socket = this.options.createSocket
      ? this.options.createSocket(this.options.url)
      : (new WebSocket(this.options.url) as unknown as SocketLike);
    this.socket = socket;
    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (event) => {
      if (typeof event.data === "string") {
        this.handleFrame(event.data);
      }
    };
    socket.onerror = () => {
      // The close handler drives reconnection.
    };
    socket.onclose = () => {
      if (this.socket !== socket) {
        return;
      }
      this.socket = null;
      this.clearTimers();
      if (this.active) {
        this.scheduleReconnect();
      }
    };
  }

  private handleFrame(raw: string): void {
    const frame = parseServerFrame(raw);
    if (!frame) {
      return;
    }
    if (frame.kind === "ready") {
      this.attempts = 0;
      const reconnected = this.hasBeenReady;
      this.hasBeenReady = true;
      this.startPing();
      if (reconnected) {
        this.options.onReconnected();
      }
    } else if (frame.kind === "pong") {
      if (this.pongTimer) {
        clearTimeout(this.pongTimer);
        this.pongTimer = null;
      }
    } else {
      this.options.onEvent(frame.event);
    }
  }

  private startPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
    }
    this.pingTimer = setInterval(() => {
      const socket = this.socket;
      if (!socket || socket.readyState !== OPEN) {
        return;
      }
      socket.send(JSON.stringify({ type: "ping" }));
      if (!this.pongTimer) {
        this.pongTimer = setTimeout(() => {
          this.pongTimer = null;
          socket.close();
        }, PONG_TIMEOUT_MS);
      }
    }, PING_INTERVAL_MS);
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer || !this.active) {
      return;
    }
    const delay = backoffDelayMs(this.attempts);
    this.attempts += 1;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect();
    }, delay);
  }
}
