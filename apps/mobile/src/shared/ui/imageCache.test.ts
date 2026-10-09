import { createImageCache } from './imageCache';

describe('image cache', () => {
  it('loads each path once and serves the cached promise', async () => {
    const loader = jest.fn(async (p: string) => `blob:${p}`);
    const cache = createImageCache(loader);
    await expect(cache.load('/photos/1/file')).resolves.toBe('blob:/photos/1/file');
    await cache.load('/photos/1/file');
    expect(loader).toHaveBeenCalledTimes(1);
  });

  it('evicts the least recently used entry beyond the limit and reports it', async () => {
    const loader = jest.fn(async (p: string) => `blob:${p}`);
    const evicted: string[] = [];
    const cache = createImageCache(loader, 2, (u) => evicted.push(u));
    await cache.load('a');
    await cache.load('b');
    await cache.load('a'); // a becomes most recent
    await cache.load('c'); // evicts b
    await Promise.resolve();
    expect(cache.size()).toBe(2);
    expect(evicted).toEqual(['blob:b']);
    await cache.load('b');
    expect(loader).toHaveBeenCalledTimes(4);
  });

  it('does not keep failed loads', async () => {
    const loader = jest.fn().mockRejectedValueOnce(new Error('x')).mockResolvedValueOnce('blob:ok');
    const cache = createImageCache(loader);
    await expect(cache.load('p')).rejects.toThrow('x');
    await Promise.resolve();
    await expect(cache.load('p')).resolves.toBe('blob:ok');
  });
});
