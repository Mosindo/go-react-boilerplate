import { QueryClient } from "@tanstack/react-query";
import { errorStatus } from "../lib/errors";

/** One shared client for the whole app. Client errors (4xx) are never retried. */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        const status = errorStatus(error);
        return (status === null || status >= 500) && failureCount < 1;
      },
      staleTime: 30_000,
      refetchOnWindowFocus: false
    },
    mutations: {
      retry: 0
    }
  }
});
