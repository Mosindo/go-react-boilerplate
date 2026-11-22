type Listener<T> = (payload: T) => void;

/** Tiny typed event bus for cross-screen UI coordination (no global state). */
function createEvent<T>() {
  const listeners = new Set<Listener<T>>();
  return {
    emit(payload: T) {
      listeners.forEach((l) => l(payload));
    },
    subscribe(listener: Listener<T>) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    }
  };
}

/** A decision taken outside the discovery deck (profile detail, likes tab, block...). */
export const profileDecided = createEvent<string>();
