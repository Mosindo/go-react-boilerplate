// Profiles the user already decided on (swipe, block) outside the discovery deck,
// e.g. from a profile page. The deck subscribes so it never shows them again.

const decided = new Set<string>();
const listeners = new Set<() => void>();

export function markDecided(userId: string): void {
  if (!decided.has(userId)) {
    decided.add(userId);
    listeners.forEach((listener) => listener());
  }
}

export function undoDecided(userId: string): void {
  decided.delete(userId);
}

export function isDecided(userId: string): boolean {
  return decided.has(userId);
}

export function subscribeDecisions(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function clearDecisions(): void {
  decided.clear();
}
