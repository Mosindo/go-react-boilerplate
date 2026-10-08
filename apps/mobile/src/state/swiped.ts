import { useEffect, useState } from "react";

// Profiles already decided in this session. The deck and the profile-detail screen both write here so the
// deck never shows a card the user has just liked or passed from another screen.
const swiped = new Set<string>();
const listeners = new Set<() => void>();

export function markSwiped(userId: string): void {
  swiped.add(userId);
  listeners.forEach((listener) => listener());
}

export function unmarkSwiped(userId: string): void {
  swiped.delete(userId);
  listeners.forEach((listener) => listener());
}

export function wasSwiped(userId: string): boolean {
  return swiped.has(userId);
}

export function clearSwiped(): void {
  swiped.clear();
}

export function useSwipedVersion(): number {
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const listener = () => setVersion((v) => v + 1);
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }, []);
  return version;
}
