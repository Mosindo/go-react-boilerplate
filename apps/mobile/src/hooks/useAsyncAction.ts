import { useCallback, useEffect, useRef, useState } from "react";

export function errorMessage(
  error: unknown,
  fallback = "Une erreur est survenue. Réessayez."
): string {
  return error instanceof Error && error.message ? error.message : fallback;
}

/**
 * Wraps an async handler with loading and error state. Overlapping calls are
 * ignored, which protects against double taps on submit buttons.
 */
export function useAsyncAction<Args extends unknown[], Result>(
  action: (...args: Args) => Promise<Result>
) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const busy = useRef(false);
  const mounted = useRef(true);
  const actionRef = useRef(action);
  actionRef.current = action;

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const run = useCallback(async (...args: Args): Promise<Result | undefined> => {
    if (busy.current) {
      return undefined;
    }
    busy.current = true;
    setLoading(true);
    setError(null);
    try {
      return await actionRef.current(...args);
    } catch (e) {
      if (mounted.current) setError(errorMessage(e));
      return undefined;
    } finally {
      busy.current = false;
      if (mounted.current) setLoading(false);
    }
  }, []);

  return { run, loading, error, setError };
}
