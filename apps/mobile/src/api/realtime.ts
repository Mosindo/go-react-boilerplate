import { API_BASE_URL } from "./client";
import { endpoints } from "./endpoints";
import type { RealtimeEvent } from "./types";

/** Opens the WebSocket with a one-time ticket (see requestSocketTicket). The only place a socket is created. */
export function openRealtimeSocket(ticket: string): WebSocket {
  const base = API_BASE_URL.replace(/^http/, "ws");
  return new WebSocket(`${base}${endpoints.realtime.socket}?ticket=${encodeURIComponent(ticket)}`);
}

export function parseRealtimeEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const event = JSON.parse(raw) as { type?: unknown; data?: unknown };
    return typeof event.type === "string" ? (event as RealtimeEvent) : null;
  } catch {
    return null;
  }
}
