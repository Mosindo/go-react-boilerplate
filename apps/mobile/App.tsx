import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { ApiError } from "./src/api/client";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { ToastHost } from "./src/shared/feedback";
import { useTheme } from "./src/theme";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      // Client errors (4xx) will not change on retry; only transient failures are retried.
      retry: (failureCount, error) => failureCount < 2 && (!(error instanceof ApiError) || error.status === 0 || error.status >= 500)
    },
    mutations: { retry: 0 }
  }
});

function Shell() {
  const palette = useTheme();
  return (
    <>
      <StatusBar style={palette.isDark ? "light" : "dark"} />
      <RootNavigator />
      <ToastHost />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <Shell />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
