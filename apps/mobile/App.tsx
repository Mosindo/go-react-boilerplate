import React from "react";
import { StatusBar } from "expo-status-bar";
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { ApiError, errorMessage } from "./src/api/client";
import { AuthProvider } from "./src/features/auth/AuthProvider";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeProvider";
import { ThemeProvider, useTheme } from "./src/theme";
import { showToast, ToastHost } from "./src/ui";

function reportGlobalError(err: unknown) {
  // Network failures and server errors get a toast; screens handle the rest.
  if (err instanceof ApiError && (err.status === 0 || err.status >= 500)) {
    showToast(errorMessage(err), "error");
  }
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: reportGlobalError }),
  mutationCache: new MutationCache({ onError: reportGlobalError }),
  defaultOptions: {
    queries: {
      retry: (count, err) => !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2,
      refetchOnWindowFocus: false,
      staleTime: 30_000
    }
  }
});

function ThemedApp() {
  const theme = useTheme();
  return (
    <>
      <StatusBar style={theme.scheme === "dark" ? "light" : "dark"} />
      <RootNavigator />
      <ToastHost />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <RealtimeProvider>
              <ThemedApp />
            </RealtimeProvider>
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
