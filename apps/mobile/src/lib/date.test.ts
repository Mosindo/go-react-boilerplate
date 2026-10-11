import type { Message } from '../api/types';
import { buildChatItems, dayKey, formatDayLabel, formatListTime, formatTime } from './date';

const NOW = new Date(2026, 2, 10, 15, 0); // 10 March 2026, 15:00 local

const at = (y: number, m: number, d: number, h: number, min: number): string =>
  new Date(y, m, d, h, min).toISOString();

const msg = (id: string, createdAt: string): Message => ({
  id,
  conversationId: 'c1',
  senderId: 'u1',
  body: id,
  createdAt,
});

describe('day labels', () => {
  it('labels today, yesterday and older days in French', () => {
    expect(formatDayLabel(at(2026, 2, 10, 8, 0), NOW)).toBe("Aujourd'hui");
    expect(formatDayLabel(at(2026, 2, 9, 23, 59), NOW)).toBe('Hier');
    expect(formatDayLabel(at(2026, 2, 1, 12, 0), NOW)).toMatch(/dimanche 1.* mars/);
    expect(formatDayLabel(at(2025, 11, 25, 12, 0), NOW)).toMatch(/25 décembre 2025/);
  });

  it('formats times and list times', () => {
    expect(formatTime(at(2026, 2, 10, 9, 5))).toBe('09:05');
    expect(formatListTime(at(2026, 2, 10, 9, 5), NOW)).toBe('09:05');
    expect(formatListTime(at(2026, 2, 9, 9, 5), NOW)).toBe('Hier');
    expect(formatListTime(at(2026, 0, 2, 9, 5), NOW)).toBe('02/01');
  });
});

describe('buildChatItems', () => {
  it('puts a day separator after the oldest message of each day (inverted order)', () => {
    // newest first
    const messages = [
      msg('d', at(2026, 2, 10, 10, 0)),
      msg('c', at(2026, 2, 10, 9, 0)),
      msg('b', at(2026, 2, 9, 22, 0)),
      msg('a', at(2026, 2, 9, 21, 0)),
    ];
    const items = buildChatItems(messages, NOW);
    expect(items.map((i) => (i.kind === 'day' ? `day:${i.label}` : i.message.id))).toEqual([
      'd',
      'c',
      "day:Aujourd'hui",
      'b',
      'a',
      'day:Hier',
    ]);
  });

  it('returns nothing for an empty conversation and one separator for a single message', () => {
    expect(buildChatItems([], NOW)).toEqual([]);
    expect(buildChatItems([msg('x', at(2026, 2, 10, 1, 0))], NOW)).toHaveLength(2);
    expect(dayKey(new Date(2026, 0, 5))).toBe('2026-01-05');
  });
});
