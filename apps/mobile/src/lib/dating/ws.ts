import type {
  AppNotification,
  ConversationSummary,
  Message,
  Photo,
  RealtimeEvent
} from "../../api/models";

export const BACKOFF_BASE_MS = 1000;
export const BACKOFF_MAX_MS = 30_000;
export const POLL_INTERVAL_MS = 30_000;

/** Convert http(s)://host[/prefix] into ws(s)://host[/prefix]/ws?ticket=… */
export function buildWsUrlFromBase(baseUrl: string, ticket: string): string {
  const trimmed = baseUrl.trim().replace(/\/+$/, "");
  const wsBase = trimmed.replace(/^http(s?):\/\//i, (_m, s: string) => `ws${s}://`);
  return `${wsBase}/ws?ticket=${encodeURIComponent(ticket)}`;
}

/**
 * Exponential backoff with jitter: the ceiling doubles from 1 s up to 30 s and the actual delay
 * is picked uniformly in [ceiling / 2, ceiling]. `attempt` starts at 0.
 */
export function computeBackoffMs(attempt: number, random: () => number = Math.random): number {
  const safeAttempt = Math.max(0, Math.min(Math.floor(attempt), 16));
  const ceiling = Math.min(BACKOFF_MAX_MS, BACKOFF_BASE_MS * 2 ** safeAttempt);
  const r = Math.min(1, Math.max(0, random()));
  return Math.round(ceiling / 2 + (ceiling / 2) * r);
}

type Rec = Record<string, unknown>;

function isRec(v: unknown): v is Rec {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function isStr(v: unknown): v is string {
  return typeof v === "string";
}

function isNullableStr(v: unknown): v is string | null {
  return v === null || typeof v === "string";
}

export function isMessage(v: unknown): v is Message {
  return (
    isRec(v) &&
    isStr(v.id) &&
    isStr(v.conversationId) &&
    isStr(v.senderId) &&
    isStr(v.body) &&
    isStr(v.createdAt) &&
    isNullableStr(v.readAt === undefined ? null : v.readAt)
  );
}

function isPhoto(v: unknown): v is Photo {
  return isRec(v) && isStr(v.id) && isStr(v.url) && typeof v.position === "number";
}

export function isConversationSummary(v: unknown): v is ConversationSummary {
  if (!isRec(v) || !isStr(v.id) || !isStr(v.matchId) || !isStr(v.matchedAt) || !isStr(v.updatedAt)) {
    return false;
  }
  if (typeof v.unreadCount !== "number") return false;
  if (!isRec(v.user) || !isStr(v.user.userId) || !isStr(v.user.firstName)) return false;
  if (!(v.user.photo === null || v.user.photo === undefined || isPhoto(v.user.photo))) return false;
  return v.lastMessage === null || v.lastMessage === undefined || isMessage(v.lastMessage);
}

export function isNotification(v: unknown): v is AppNotification {
  return (
    isRec(v) &&
    isStr(v.id) &&
    (v.type === "match" || v.type === "message") &&
    isStr(v.title) &&
    isStr(v.body) &&
    isRec(v.data) &&
    typeof v.isRead === "boolean" &&
    isStr(v.createdAt)
  );
}

/** Normalise optional nulls so downstream code can rely on the declared wire types. */
function normaliseConversation(c: ConversationSummary): ConversationSummary {
  return {
    ...c,
    lastMessage: c.lastMessage ? { ...c.lastMessage, readAt: c.lastMessage.readAt ?? null } : null,
    user: { ...c.user, age: c.user.age ?? null, photo: c.user.photo ?? null }
  };
}

/**
 * Parse a raw WebSocket frame. Returns null for malformed frames and unknown event types
 * (forward compatibility: new server events must never crash the client).
 */
export function parseRealtimeEvent(raw: unknown): RealtimeEvent | null {
  let value: unknown = raw;
  if (typeof raw === "string") {
    try {
      value = JSON.parse(raw);
    } catch {
      return null;
    }
  }
  if (!isRec(value) || !isStr(value.type) || !isRec(value.data)) return null;
  const data = value.data;
  switch (value.type) {
    case "message.new":
      return isMessage(data.message)
        ? {
            type: "message.new",
            data: { message: { ...data.message, readAt: data.message.readAt ?? null } }
          }
        : null;
    case "conversation.read":
      return isStr(data.conversationId) && isStr(data.readAt)
        ? { type: "conversation.read", data: { conversationId: data.conversationId, readAt: data.readAt } }
        : null;
    case "match.new":
      return isConversationSummary(data.conversation)
        ? { type: "match.new", data: { conversation: normaliseConversation(data.conversation) } }
        : null;
    case "match.removed":
      return isStr(data.conversationId)
        ? { type: "match.removed", data: { conversationId: data.conversationId } }
        : null;
    case "notification.new":
      return isNotification(data.notification)
        ? { type: "notification.new", data: { notification: data.notification } }
        : null;
    default:
      return null;
  }
}
