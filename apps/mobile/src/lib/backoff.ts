/** Delai de reconnexion exponentiel plafonne (en ms): 1s, 2s, 4s ... 30s. */
export function backoffDelay(attempt: number, base = 1000, max = 30_000): number {
  return Math.min(max, base * 2 ** Math.max(0, attempt));
}
