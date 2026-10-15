// Tiny pub/sub so a screen that hides a person (block, report) can tell the
// discovery deck to drop that card without prop drilling through navigation.

type Listener = (userId: string) => void;

const listeners = new Set<Listener>();

export function onProfileRemoved(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function announceProfileRemoved(userId: string): void {
  listeners.forEach((listener) => listener(userId));
}
