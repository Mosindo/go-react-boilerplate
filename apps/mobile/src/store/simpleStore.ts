import { useSyncExternalStore } from 'react';

/** Minimal external store (no dependency) for small pieces of cross-screen UI state. */
export function createStore<T>(initial: T) {
  let value = initial;
  const listeners = new Set<() => void>();
  return {
    get: () => value,
    set(next: T) {
      if (Object.is(next, value)) return;
      value = next;
      listeners.forEach((l) => l());
    },
    subscribe(l: () => void) {
      listeners.add(l);
      return () => {
        listeners.delete(l);
      };
    },
    use(): T {
      return useSyncExternalStore(
        (l) => {
          listeners.add(l);
          return () => {
            listeners.delete(l);
          };
        },
        () => value,
        () => value,
      );
    },
  };
}
