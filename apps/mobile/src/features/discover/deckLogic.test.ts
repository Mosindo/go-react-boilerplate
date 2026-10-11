import type { PublicProfile } from '../../api/types';
import { deckReducer, initialDeck, shouldPrefetch } from './deckLogic';

const profile = (id: string): PublicProfile => ({
  userId: id,
  firstName: id,
  age: 30,
  gender: 'woman',
  bio: '',
  city: 'Lyon',
  interests: [],
  photos: [],
  distanceKm: null,
});

describe('deckReducer', () => {
  it('queues a batch and dedupes against what is already queued', () => {
    let s = deckReducer(initialDeck, {
      type: 'batch',
      profiles: [profile('a'), profile('b'), profile('a')],
    });
    expect(s.queue.map((p) => p.userId)).toEqual(['a', 'b']);
    s = deckReducer(s, { type: 'batch', profiles: [profile('b'), profile('c')] });
    expect(s.queue.map((p) => p.userId)).toEqual(['a', 'b', 'c']);
    expect(s.exhausted).toBe(false);
  });

  it('never re-adds someone already swiped, and flags exhaustion on an all-duplicate batch', () => {
    let s = deckReducer(initialDeck, { type: 'batch', profiles: [profile('a')] });
    s = deckReducer(s, { type: 'remove', userId: 'a' });
    expect(s.queue).toHaveLength(0);
    s = deckReducer(s, { type: 'batch', profiles: [profile('a')] });
    expect(s.queue).toHaveLength(0);
    expect(s.exhausted).toBe(true);
  });

  it('an empty batch means no one is left', () => {
    expect(deckReducer(initialDeck, { type: 'batch', profiles: [] }).exhausted).toBe(true);
  });

  it('restores a failed swipe on top and resumes fetching', () => {
    let s = deckReducer(initialDeck, { type: 'batch', profiles: [profile('a'), profile('b')] });
    s = deckReducer(s, { type: 'remove', userId: 'a' });
    s = deckReducer({ ...s, exhausted: true }, { type: 'restore', profile: profile('a') });
    expect(s.queue.map((p) => p.userId)).toEqual(['a', 'b']);
    expect(s.exhausted).toBe(false);
    expect(deckReducer(s, { type: 'restore', profile: profile('a') })).toBe(s);
  });

  it('resume clears exhaustion without touching the queue', () => {
    const s = deckReducer({ ...initialDeck, exhausted: true }, { type: 'resume' });
    expect(s.exhausted).toBe(false);
  });
});

describe('shouldPrefetch', () => {
  const withCards = (n: number) => ({
    ...initialDeck,
    queue: Array.from({ length: n }, (_, i) => profile(String(i))),
  });

  it('prefetches when 3 or fewer cards remain', () => {
    expect(shouldPrefetch(withCards(4), false)).toBe(false);
    expect(shouldPrefetch(withCards(3), false)).toBe(true);
    expect(shouldPrefetch(withCards(0), false)).toBe(true);
  });

  it('does not fetch while busy or after exhaustion', () => {
    expect(shouldPrefetch(withCards(1), true)).toBe(false);
    expect(shouldPrefetch({ ...withCards(1), exhausted: true }, false)).toBe(false);
  });
});
