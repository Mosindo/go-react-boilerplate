import type { RealtimeEvent } from '../api/types';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

const str = (v: unknown): boolean => typeof v === 'string';

/**
 * Parses a raw WebSocket frame. Returns null for pong, unknown types or malformed payloads
 * so a bad frame can never crash the app.
 */
export function parseFrame(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== 'string') return null;
  let frame: unknown;
  try {
    frame = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(frame) || typeof frame.type !== 'string' || !isRecord(frame.data)) return null;
  const data = frame.data;
  switch (frame.type) {
    case 'message.new':
      return str(data.id) && str(data.conversationId) && str(data.senderId)
        ? ({ type: 'message.new', data } as unknown as RealtimeEvent)
        : null;
    case 'message.read':
      return str(data.conversationId) && str(data.readerId) && str(data.readAt)
        ? ({ type: 'message.read', data } as unknown as RealtimeEvent)
        : null;
    case 'match.new':
      return str(data.id) && isRecord(data.user)
        ? ({ type: 'match.new', data } as unknown as RealtimeEvent)
        : null;
    case 'notification.new':
      return str(data.id) && str(data.type)
        ? ({ type: 'notification.new', data } as unknown as RealtimeEvent)
        : null;
    case 'conversation.removed':
      return str(data.conversationId)
        ? ({ type: 'conversation.removed', data } as unknown as RealtimeEvent)
        : null;
    default:
      return null;
  }
}
