// Queue logic for the discovery deck, kept free of React so it is easy to test.

export const DECK_REFILL_THRESHOLD = 3;
export const DECK_BATCH_SIZE = 10;
/** The API accepts at most 50 ids in `exclude`. */
export const MAX_EXCLUDE = 50;

export type HasUserId = { userId: string };

/** Appends fresh cards while dropping anything already queued or already decided. */
export function mergeCards<T extends HasUserId>(queue: T[], incoming: T[], decided: ReadonlySet<string>): T[] {
  const seen = new Set(queue.map((card) => card.userId));
  const added: T[] = [];
  for (const card of incoming) {
    if (!seen.has(card.userId) && !decided.has(card.userId)) {
      seen.add(card.userId);
      added.push(card);
    }
  }
  return added.length === 0 ? queue : [...queue, ...added];
}

export function shouldRefill(queueLength: number, exhausted: boolean, loading: boolean): boolean {
  return !exhausted && !loading && queueLength <= DECK_REFILL_THRESHOLD;
}

/** Ids sent as `exclude` so the server never repeats what the client already holds. */
export function excludeIds<T extends HasUserId>(queue: T[]): string[] {
  return queue.slice(0, MAX_EXCLUDE).map((card) => card.userId);
}
