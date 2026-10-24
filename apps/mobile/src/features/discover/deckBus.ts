type Listener = (userId: string) => void;
const listeners = new Set<Listener>();

/** Lets screens outside the deck (profile detail) tell it that a profile was already handled. */
export function emitHandled(userId: string): void {
  listeners.forEach((l) => l(userId));
}

export function onHandled(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
