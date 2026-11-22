import { API_BASE_URL, getFreshAccessToken } from "./client";

export type RealtimeEvent = {
  type: string;
  data: unknown;
  at: string;
};

type Handlers = {
  onEvent: (event: RealtimeEvent) => void;
  onStatus?: (connected: boolean) => void;
};

/**
 * Minimal Server-Sent Events client built on XMLHttpRequest streaming, which
 * React Native supports on iOS and Android (EventSource does not exist there
 * and cannot send an Authorization header anyway). Reconnects with
 * exponential backoff and always uses a fresh access token.
 */
export function connectRealtime({ onEvent, onStatus }: Handlers): () => void {
  let xhr: XMLHttpRequest | null = null;
  let stopped = false;
  let attempt = 0;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;

  const scheduleReconnect = (immediate = false) => {
    if (stopped) {
      return;
    }
    onStatus?.(false);
    const delay = immediate ? 250 : Math.min(30_000, 1000 * 2 ** attempt) + Math.random() * 500;
    attempt = Math.min(attempt + 1, 5);
    retryTimer = setTimeout(open, delay);
  };

  const open = async () => {
    if (stopped) {
      return;
    }
    const token = await getFreshAccessToken();
    if (!token || stopped) {
      scheduleReconnect();
      return;
    }
    let cursor = 0;
    let buffer = "";
    let reconnectRequested = false;
    const request = new XMLHttpRequest();
    xhr = request;
    request.open("GET", `${API_BASE_URL}/events`);
    request.setRequestHeader("Authorization", `Bearer ${token}`);
    request.setRequestHeader("Accept", "text/event-stream");
    request.timeout = 0;

    const consume = () => {
      const text = request.responseText ?? "";
      if (text.length <= cursor) {
        return;
      }
      buffer += text.slice(cursor);
      cursor = text.length;
      let boundary = buffer.indexOf("\n\n");
      while (boundary !== -1) {
        const chunk = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        boundary = buffer.indexOf("\n\n");
        let eventName = "message";
        const dataLines: string[] = [];
        for (const line of chunk.split("\n")) {
          if (line.startsWith("event:")) {
            eventName = line.slice(6).trim();
          } else if (line.startsWith("data:")) {
            dataLines.push(line.slice(5).trimStart());
          }
        }
        if (eventName === "reconnect") {
          reconnectRequested = true;
          continue;
        }
        if (dataLines.length === 0) {
          continue;
        }
        try {
          onEvent(JSON.parse(dataLines.join("\n")) as RealtimeEvent);
        } catch {
          // Ignore malformed frames.
        }
      }
    };

    request.onreadystatechange = () => {
      if (request.readyState === XMLHttpRequest.HEADERS_RECEIVED) {
        if (request.status === 200) {
          attempt = 0;
          onStatus?.(true);
        }
      }
      if (request.readyState === XMLHttpRequest.LOADING || request.readyState === XMLHttpRequest.DONE) {
        consume();
      }
      if (request.readyState === XMLHttpRequest.DONE && xhr === request) {
        xhr = null;
        scheduleReconnect(reconnectRequested && request.status === 200);
      }
    };
    request.onprogress = consume;
    request.send();
  };

  void open();

  return () => {
    stopped = true;
    if (retryTimer) {
      clearTimeout(retryTimer);
    }
    const current = xhr;
    xhr = null;
    current?.abort();
    onStatus?.(false);
  };
}
