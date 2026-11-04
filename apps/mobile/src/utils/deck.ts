export type HasUserId = { userId: string };

export const PREFETCH_THRESHOLD = 3;

/** Appends incoming profiles, dropping anyone already queued or already judged. */
export function mergeDeck<T extends HasUserId>(deck: T[], incoming: T[], seen: ReadonlySet<string>): T[] {
  const known = new Set(deck.map((p) => p.userId));
  const out = [...deck];
  for (const p of incoming) {
    if (known.has(p.userId) || seen.has(p.userId)) continue;
    known.add(p.userId);
    out.push(p);
  }
  return out;
}

export function shouldPrefetch(remaining: number, loading: boolean, exhausted: boolean): boolean {
  return !loading && !exhausted && remaining <= PREFETCH_THRESHOLD;
}
