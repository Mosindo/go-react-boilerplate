import { isOwnSwipeMatch, swipeFinished, swipeStarted } from './swipeGuard';

describe('swipeGuard', () => {
  it('covers a swipe in flight and the grace period after a match', () => {
    swipeStarted('u1');
    expect(isOwnSwipeMatch('u1')).toBe(true);
    swipeFinished('u1', true);
    expect(isOwnSwipeMatch('u1')).toBe(true);
    expect(isOwnSwipeMatch('u1', Date.now() + 11_000)).toBe(false);
  });

  it('forgets swipes that did not match', () => {
    swipeStarted('u2');
    swipeFinished('u2', false);
    expect(isOwnSwipeMatch('u2')).toBe(false);
    expect(isOwnSwipeMatch('never-swiped')).toBe(false);
  });
});
