import { API_BASE_URL, refreshTokens } from "./client";
import { getTokens } from "./session";

export type RealtimeEvent = {
  type: string;
  data?: unknown;
  at: string;
};

type Handler = (event: RealtimeEvent) => void;

const TOKEN_EXPIRED = 4001;
const UNAUTHORIZED = 4003;

/**
 * Resilient WebSocket client: authenticates with the first frame (tokens are
 * never put in the URL), refreshes the access token when the server closes
 * with 4001 and reconnects with exponential backoff.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private handler: Handler;
  private attempts = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  constructor(handler: Handler) {
    this.handler = handler;
  }

  start() {
    if (!this.stopped) {
      return;
    }
    this.stopped = false;
    this.connect();
  }

  stop() {
    this.stopped = true;
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
  }

  /** Reconnect immediately (e.g. when the app returns to the foreground). */
  wake() {
    if (this.stopped) {
      return;
    }
    if (!this.socket || this.socket.readyState === WebSocket.CLOSED) {
      this.attempts = 0;
      this.connect();
    }
  }

  private connect() {
    if (this.stopped || !API_BASE_URL) {
      return;
    }
    const token = getTokens()?.accessToken;
    if (!token) {
      return;
    }
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    const socket = new WebSocket(`${API_BASE_URL.replace(/^http/, "ws")}/realtime`);
    this.socket = socket;

    socket.onopen = () => {
      socket.send(JSON.stringify({ type: "auth", token }));
    };
    socket.onmessage = (message) => {
      try {
        const event = JSON.parse(String(message.data)) as RealtimeEvent;
        if (event.type === "ready") {
          this.attempts = 0;
        }
        this.handler(event);
      } catch {
        // Ignore malformed frames.
      }
    };
    socket.onclose = (event) => {
      if (this.socket !== socket || this.stopped) {
        return;
      }
      this.socket = null;
      if (event.code === TOKEN_EXPIRED || event.code === UNAUTHORIZED) {
        void refreshTokens().then((tokens) => {
          if (tokens) {
            this.schedule(250);
          }
        });
        return;
      }
      this.schedule(Math.min(30_000, 1_000 * 2 ** this.attempts));
      this.attempts += 1;
    };
  }

  private schedule(delay: number) {
    if (this.stopped) {
      return;
    }
    this.timer = setTimeout(() => this.connect(), delay);
  }
}
