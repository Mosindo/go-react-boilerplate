import { useSyncExternalStore } from "react";

let epoch = 0;
const listeners = new Set<() => void>();

/** Forces the discovery deck to start over (after preferences or location changed). */
export function bumpDeckEpoch(): void {
  epoch += 1;
  listeners.forEach((listener) => listener());
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useDeckEpoch(): number {
  return useSyncExternalStore(subscribe, () => epoch);
}
