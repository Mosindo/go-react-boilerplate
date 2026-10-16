/**
 * Wraps an async function so concurrent callers share one in-flight run.
 * Used for token refresh: refresh tokens rotate, so two parallel refreshes would
 * invalidate each other.
 */
export function singleFlight<T>(run: () => Promise<T>): () => Promise<T> {
  let inflight: Promise<T> | null = null;
  return () => {
    if (!inflight) {
      inflight = run().finally(() => {
        inflight = null;
      });
    }
    return inflight;
  };
}
