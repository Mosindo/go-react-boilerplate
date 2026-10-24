import React, { useState } from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { ApiError } from "./src/api/client";
import { AuthProvider } from "./src/auth/AuthProvider";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeProvider";
import { ThemeProvider, useTheme } from "./src/theme/theme";
import { DialogProvider, ToastProvider } from "./src/ui";

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        // Client errors (4xx) will not fix themselves: only retry network/server failures.
        retry: (count, error) =>
          !(error instanceof ApiError && error.status >= 400 && error.status < 500) && count < 2,
        refetchOnWindowFocus: false,
      },
    },
  });
}

function Shell() {
  const { scheme } = useTheme();
  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      <DialogProvider>
        <ToastProvider>
          <AuthProvider>
            <RealtimeProvider>
              <RootNavigator />
            </RealtimeProvider>
          </AuthProvider>
        </ToastProvider>
      </DialogProvider>
    </>
  );
}

export default function App() {
  const [queryClient] = useState(createQueryClient);
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <Shell />
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
