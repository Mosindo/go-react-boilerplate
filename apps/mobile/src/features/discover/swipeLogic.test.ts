import { decideSwipe, rotationDeg } from './swipeLogic';

const W = 400;

describe('decideSwipe', () => {
  it('commits past the distance threshold in either direction', () => {
    expect(decideSwipe(120, 0, W)).toBe('like');
    expect(decideSwipe(-120, 0, W)).toBe('pass');
  });

  it('snaps back for small, slow drags', () => {
    expect(decideSwipe(60, 0.1, W)).toBeNull();
    expect(decideSwipe(-100, 0.2, W)).toBeNull();
    expect(decideSwipe(0, 0, W)).toBeNull();
  });

  it('commits on a fast flick that travelled a minimum distance', () => {
    expect(decideSwipe(50, 0.9, W)).toBe('like');
    expect(decideSwipe(-50, -0.9, W)).toBe('pass');
  });

  it('ignores flicks that are too short or against the drag direction', () => {
    expect(decideSwipe(10, 1.5, W)).toBeNull();
    expect(decideSwipe(60, -1.5, W)).toBeNull();
  });

  it('is safe before layout (zero width)', () => {
    expect(decideSwipe(500, 2, 0)).toBeNull();
  });
});

describe('rotationDeg', () => {
  it('scales with the drag and clamps', () => {
    expect(rotationDeg(0, W)).toBe(0);
    expect(rotationDeg(100, W)).toBeGreaterThan(0);
    expect(rotationDeg(-100, W)).toBeLessThan(0);
    expect(rotationDeg(10_000, W)).toBe(14);
    expect(rotationDeg(-10_000, W)).toBe(-14);
  });
});
