/** Small LRU cache of authenticated image downloads turned into blob URLs (web only). */
export function createImageCache(
  loader: (path: string) => Promise<string>,
  maxEntries = 120,
  onEvict?: (url: string) => void,
) {
  const entries = new Map<string, Promise<string>>();
  return {
    load(path: string): Promise<string> {
      const hit = entries.get(path);
      if (hit) {
        entries.delete(path);
        entries.set(path, hit);
        return hit;
      }
      const promise = loader(path);
      entries.set(path, promise);
      promise.catch(() => {
        if (entries.get(path) === promise) entries.delete(path);
      });
      while (entries.size > maxEntries) {
        const oldest = entries.keys().next().value as string;
        const evicted = entries.get(oldest);
        entries.delete(oldest);
        if (onEvict && evicted) void evicted.then(onEvict, () => undefined);
      }
      return promise;
    },
    invalidate(path: string) {
      entries.delete(path);
    },
    size: () => entries.size,
    clear() {
      entries.clear();
    },
  };
}
