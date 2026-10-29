import { useSyncExternalStore } from "react";

/**
 * Tiny shared store that lets screens talk to the discovery deck without
 * passing callbacks through navigation params.
 *  - version: bumped when preferences/profile change, so the deck reloads
 *  - resolved: profiles already liked/passed elsewhere (e.g. from a profile page)
 */
type Snapshot = { version: number; resolved: ReadonlySet<string> };

let snapshot: Snapshot = { version: 0, resolved: new Set() };
const listeners = new Set<() => void>();

function update(next: Snapshot): void {
  snapshot = next;
  listeners.forEach((listener) => listener());
}

export function bumpDiscoverVersion(): void {
  update({ version: snapshot.version + 1, resolved: new Set() });
}

export function markProfileResolved(userId: string): void {
  update({ version: snapshot.version, resolved: new Set(snapshot.resolved).add(userId) });
}

export function unmarkProfileResolved(userId: string): void {
  const next = new Set(snapshot.resolved);
  next.delete(userId);
  update({ version: snapshot.version, resolved: next });
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useDiscoverStore(): Snapshot {
  return useSyncExternalStore(
    subscribe,
    () => snapshot,
    () => snapshot
  );
}

export function resetDiscoverStore(): void {
  update({ version: 0, resolved: new Set() });
}
