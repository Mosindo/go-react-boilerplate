import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthProvider";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { ToastHost } from "./src/ui";
import { isApiError } from "./src/api/client";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Never retry client errors (403/404/409...) — only transient failures.
      retry: (failureCount, error) => failureCount < 2 && (!isApiError(error) || error.status === 0 || error.status >= 500),
      staleTime: 30_000,
      refetchOnWindowFocus: false
    },
    mutations: { retry: 0 }
  }
});

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <StatusBar style="light" />
          <RootNavigator />
          <ToastHost />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
