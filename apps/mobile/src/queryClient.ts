import { MutationCache, QueryClient } from "@tanstack/react-query";
import { ApiError, errorMessage } from "./api";
import { showToast } from "./shared/feedback";

function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status >= 400 && error.status < 500) {
    return false;
  }
  return failureCount < 2;
}

export const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      // Forms that render their own inline error opt out with `meta: { silent: true }`.
      if (mutation.meta?.silent !== true) {
        showToast(errorMessage(error), "error");
      }
    }
  }),
  defaultOptions: {
    queries: { retry: shouldRetry, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});
