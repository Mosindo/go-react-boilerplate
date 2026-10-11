export const BACKOFF_BASE_MS = 1_000;
export const BACKOFF_MAX_MS = 30_000;

/** Exponential backoff with +/-25% jitter. `random` is injectable for tests (0..1). */
export function backoffDelay(attempt: number, random: () => number = Math.random): number {
  const exp = Math.min(BACKOFF_MAX_MS, BACKOFF_BASE_MS * 2 ** Math.max(0, attempt));
  const jitter = 0.75 + random() * 0.5;
  return Math.min(BACKOFF_MAX_MS, Math.round(exp * jitter));
}
