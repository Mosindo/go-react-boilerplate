type Listener = (userId: string) => void;

const listeners = new Set<Listener>();

/** Lets a profile opened from the deck tell the deck it has been handled. */
export function emitSwiped(userId: string): void {
  listeners.forEach((listener) => listener(userId));
}

export function subscribeSwiped(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
