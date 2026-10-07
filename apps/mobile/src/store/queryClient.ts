import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";

/** Single shared QueryClient. Never retry client errors (4xx); retry transient failures once. */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) =>
        failureCount < 1 && !(error instanceof ApiError && error.status >= 400 && error.status < 500),
      staleTime: 30_000,
      refetchOnWindowFocus: false
    },
    mutations: { retry: 0 }
  }
});
