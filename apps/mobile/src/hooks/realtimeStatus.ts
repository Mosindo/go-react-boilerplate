import { useSyncExternalStore } from "react";

let connected = false;
const listeners = new Set<() => void>();

export function setRealtimeConnected(value: boolean) {
  if (connected !== value) {
    connected = value;
    listeners.forEach((l) => l());
  }
}

export function useRealtimeStatus(): boolean {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => connected,
    () => connected
  );
}
