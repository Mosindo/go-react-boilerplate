import { backoffDelay } from "../lib/backoff";

export type RealtimeEvent = { type: string; payload: unknown };
type Listener = (event: RealtimeEvent) => void;

/**
 * Client WebSocket avec reconnexion exponentielle. Le token n'est jamais place
 * dans l'URL: il est envoye dans un premier message {"type":"auth"}.
 */
export class RealtimeClient {
  private socket: WebSocket | null = null;
  private listeners = new Set<Listener>();
  private attempts = 0;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private stopped = true;

  constructor(
    private readonly url: string,
    /** Renvoie un access token valide (renouvele si necessaire). */
    private readonly getToken: () => Promise<string | null>,
  ) {}

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  start(): void {
    if (!this.stopped) {
      return;
    }
    this.stopped = false;
    void this.connect();
  }

  /** Force une reconnexion immediate (ex: retour au premier plan). */
  reconnectNow(): void {
    if (this.stopped) {
      return;
    }
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      return;
    }
    this.clearTimer();
    void this.connect();
  }

  stop(): void {
    this.stopped = true;
    this.clearTimer();
    if (this.socket) {
      this.socket.onclose = null;
      this.socket.close();
      this.socket = null;
    }
  }

  private clearTimer(): void {
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.stopped) {
      return;
    }
    this.clearTimer();
    this.timer = setTimeout(() => {
      void this.connect();
    }, backoffDelay(this.attempts));
    this.attempts += 1;
  }

  private async connect(): Promise<void> {
    if (this.stopped) {
      return;
    }
    const token = await this.getToken();
    if (!token || this.stopped) {
      this.scheduleReconnect();
      return;
    }
    const socket = new WebSocket(this.url);
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
        this.listeners.forEach((listener) => listener(event));
      } catch {
        // message illisible: ignore
      }
    };
    socket.onerror = () => {
      // onclose suit toujours; la reconnexion y est geree
    };
    socket.onclose = () => {
      if (this.socket === socket) {
        this.socket = null;
      }
      this.scheduleReconnect();
    };
  }
}
