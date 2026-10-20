import type { Profile } from "../api/types";

export function appendProfiles(
  queue: Profile[],
  incoming: Profile[],
  handled: ReadonlySet<string>
): Profile[] {
  const known = new Set(queue.map((p) => p.userId));
  const fresh = incoming.filter((p) => !known.has(p.userId) && !handled.has(p.userId));
  return fresh.length > 0 ? [...queue, ...fresh] : queue;
}
