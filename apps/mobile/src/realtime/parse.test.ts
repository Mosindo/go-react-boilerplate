import { backoffDelay } from './backoff';
import { parseFrame } from './parse';

describe('parseFrame', () => {
  it('parses the documented frames', () => {
    expect(
      parseFrame(
        JSON.stringify({
          type: 'message.new',
          data: { id: 'm', conversationId: 'c', senderId: 's', body: 'x', createdAt: 't' },
        }),
      ),
    ).toMatchObject({ type: 'message.new' });
    expect(
      parseFrame(
        JSON.stringify({
          type: 'message.read',
          data: { conversationId: 'c', readerId: 'r', readAt: 't' },
        }),
      ),
    ).toMatchObject({ type: 'message.read' });
    expect(
      parseFrame(JSON.stringify({ type: 'conversation.removed', data: { conversationId: 'c' } })),
    ).toEqual({
      type: 'conversation.removed',
      data: { conversationId: 'c' },
    });
  });

  it('ignores pong, unknown types and malformed payloads', () => {
    expect(parseFrame(JSON.stringify({ type: 'pong', data: {} }))).toBeNull();
    expect(parseFrame(JSON.stringify({ type: 'wat', data: {} }))).toBeNull();
    expect(parseFrame(JSON.stringify({ type: 'message.new', data: { id: 1 } }))).toBeNull();
    expect(parseFrame('not json')).toBeNull();
    expect(parseFrame(42)).toBeNull();
    expect(parseFrame(JSON.stringify({ type: 'match.new' }))).toBeNull();
  });
});

describe('backoffDelay', () => {
  const mid = () => 0.5; // jitter factor 1.0
  it('doubles from 1s and caps at 30s', () => {
    expect([0, 1, 2, 3, 4, 5, 6, 10].map((n) => backoffDelay(n, mid))).toEqual([
      1000, 2000, 4000, 8000, 16000, 30000, 30000, 30000,
    ]);
  });

  it('applies +/-25% jitter', () => {
    expect(backoffDelay(2, () => 0)).toBe(3000);
    expect(backoffDelay(2, () => 1)).toBe(5000);
  });
});
