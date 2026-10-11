import type { Photo } from '../../api/types';
import { movePhoto } from './photoOrder';

const photos: Photo[] = ['a', 'b', 'c'].map((id, i) => ({
  id,
  position: i,
  isPrimary: i === 0,
  url: `/p/${id}`,
}));

describe('movePhoto', () => {
  it('moves to first and renumbers, making it primary', () => {
    const next = movePhoto(photos, 2, 0);
    expect(next.map((p) => p.id)).toEqual(['c', 'a', 'b']);
    expect(next.map((p) => p.position)).toEqual([0, 1, 2]);
    expect(next.map((p) => p.isPrimary)).toEqual([true, false, false]);
  });

  it('moves one step and ignores out-of-range moves', () => {
    expect(movePhoto(photos, 0, 1).map((p) => p.id)).toEqual(['b', 'a', 'c']);
    expect(movePhoto(photos, 0, -1)).toBe(photos);
    expect(movePhoto(photos, 2, 3)).toBe(photos);
    expect(movePhoto(photos, 1, 1)).toBe(photos);
  });
});
