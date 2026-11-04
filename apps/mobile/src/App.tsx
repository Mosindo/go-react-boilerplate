import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { isApiError } from "./api/client";
import { SessionProvider } from "./hooks/session";
import { RootNavigator } from "./navigation/RootNavigator";
import { ToastProvider } from "./shared/ui";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      // Do not hammer the API on client errors (400-499); retry transient failures twice.
      retry: (count, err) => !(isApiError(err) && err.status >= 400 && err.status < 500) && count < 2,
    },
  },
});

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <SessionProvider>
            <StatusBar style="auto" />
            <RootNavigator />
          </SessionProvider>
        </ToastProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
