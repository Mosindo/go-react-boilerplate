// Tiny event channel so editing preferences can tell the (still mounted)
// Discover tab to throw away its buffered candidates and refill.
type Listener = () => void;
const listeners = new Set<Listener>();

export function onDiscoverReset(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function emitDiscoverReset(): void {
  listeners.forEach((listener) => listener());
}
