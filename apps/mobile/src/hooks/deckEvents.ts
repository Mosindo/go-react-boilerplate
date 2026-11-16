type Listener = (userId: string) => void;
const listeners = new Set<Listener>();

/** Notifies the discovery deck that a profile was answered elsewhere (profile detail screen). */
export function emitDeckDecision(userId: string) {
  listeners.forEach((l) => l(userId));
}

export function onDeckDecision(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
