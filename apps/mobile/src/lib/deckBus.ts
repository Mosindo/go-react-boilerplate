// Tiny event bus: the profile detail screen reports a like/pass so the discovery deck,
// which keeps its own queue, can drop that card without refetching.
type Listener = (profileId: string) => void;
const listeners = new Set<Listener>();

export function onProfileDecided(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function emitProfileDecided(profileId: string): void {
  listeners.forEach((l) => l(profileId));
}
