import type { PublicProfile } from "../api/types";

/** Merges a fresh batch, dropping profiles already shown or queued. */
export function mergeBatch(
  queue: PublicProfile[],
  incoming: PublicProfile[],
  alreadySeen: ReadonlySet<string>
): PublicProfile[] {
  const queued = new Set(queue.map((p) => p.userId));
  const fresh = incoming.filter((p) => !queued.has(p.userId) && !alreadySeen.has(p.userId));
  return [...queue, ...fresh];
}
