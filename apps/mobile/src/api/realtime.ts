import { api } from "./client";
import { realtimeUrl } from "./config";
import type { AppNotification, Match, Message, ReadReceipt } from "./types";

export type RealtimeEvent =
  | { type: "ready"; data: null }
  | { type: "message.created"; data: Message }
  | { type: "conversation.read"; data: ReadReceipt }
  | { type: "match.created"; data: Match }
  | { type: "match.removed"; data: { matchId: string; conversationId: string } }
  | { type: "notification.created"; data: AppNotification };

export async function openRealtimeSocket(): Promise<WebSocket> {
  const { ticket } = await api<{ ticket: string }>("/realtime/ticket", { method: "POST" });
  return new WebSocket(realtimeUrl(ticket));
}

export function parseEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { type?: unknown };
    return typeof parsed.type === "string" ? (parsed as RealtimeEvent) : null;
  } catch {
    return null;
  }
}
