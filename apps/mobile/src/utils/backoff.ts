/** Exponential backoff with full jitter: delay in [base*2^n/2, base*2^n], capped. */
export function backoffDelay(attempt: number, random: () => number = Math.random, baseMs = 1000, capMs = 30_000): number {
  const ceiling = Math.min(capMs, baseMs * 2 ** Math.max(0, attempt));
  return Math.round(ceiling / 2 + (ceiling / 2) * random());
}
