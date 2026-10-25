import { useSyncExternalStore } from "react";
import type { MatchItem } from "../../api";

let current: MatchItem | null = null;
const announced = new Set<string>();
const listeners = new Set<() => void>();

function emit(): void {
  listeners.forEach((listener) => listener());
}

/** Opens the celebration once per match, whichever of the swipe response or the socket event arrives first. */
export function announceMatch(match: MatchItem): void {
  if (announced.has(match.matchId)) {
    return;
  }
  announced.add(match.matchId);
  current = match;
  emit();
}

export function dismissMatch(): void {
  current = null;
  emit();
}

export function resetMatchAnnouncements(): void {
  announced.clear();
  current = null;
  emit();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useAnnouncedMatch(): MatchItem | null {
  return useSyncExternalStore(subscribe, () => current);
}
