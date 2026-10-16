// Keeps the discovery deck free of duplicates: the server never re-serves a swiped
// profile, but a refetch can return profiles that are already queued locally.

export type Identified = { id: string };

export function mergeQueue<T extends Identified>(current: readonly T[], incoming: readonly T[], seen: ReadonlySet<string>): T[] {
  const known = new Set(current.map((item) => item.id));
  const merged = [...current];
  for (const item of incoming) {
    if (!known.has(item.id) && !seen.has(item.id)) {
      known.add(item.id);
      merged.push(item);
    }
  }
  return merged;
}

export function needsRefill(queueLength: number, threshold = 3): boolean {
  return queueLength <= threshold;
}
