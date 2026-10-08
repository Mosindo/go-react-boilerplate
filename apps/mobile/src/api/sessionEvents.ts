type Listener = () => void;

const expiredListeners = new Set<Listener>();

/** Fired when a request cannot be authenticated even after a token refresh. */
export function onSessionExpired(listener: Listener): () => void {
  expiredListeners.add(listener);
  return () => {
    expiredListeners.delete(listener);
  };
}

export function emitSessionExpired(): void {
  expiredListeners.forEach((listener) => listener());
}
