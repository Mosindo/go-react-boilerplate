import type { RealtimeEvent } from "../api";

export type ServerFrame = { kind: "ready" } | { kind: "pong" } | { kind: "event"; event: RealtimeEvent };

const EVENT_TYPES = new Set<RealtimeEvent["type"]>([
  "message",
  "read",
  "match",
  "conversation_removed",
  "notification"
]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/** Parses one text frame from the server; unknown or malformed frames yield null. */
export function parseServerFrame(raw: unknown): ServerFrame | null {
  if (typeof raw !== "string") {
    return null;
  }
  let frame: unknown;
  try {
    frame = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(frame) || typeof frame.type !== "string") {
    return null;
  }
  if (frame.type === "ready") return { kind: "ready" };
  if (frame.type === "pong") return { kind: "pong" };
  if (EVENT_TYPES.has(frame.type as RealtimeEvent["type"]) && isRecord(frame.data)) {
    // The payload shapes are guaranteed by docs/API.md; the discriminant was checked above.
    return { kind: "event", event: frame as unknown as RealtimeEvent };
  }
  return null;
}

export function toWebSocketUrl(baseUrl: string): string {
  return `${baseUrl.replace(/^http/, "ws")}/ws`;
}
