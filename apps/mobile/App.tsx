import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import React, { useState } from "react";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { isApiError } from "./src/api/client";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { useThemeMode } from "./src/theme";
import { ToastHost } from "./src/ui";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 15_000,
        retry: (failureCount, error) =>
          // Client errors will not fix themselves; transient ones get two retries.
          !(isApiError(error) && error.status >= 400 && error.status < 500) && failureCount < 2
      }
    }
  });
}

export default function App() {
  const [client] = useState(createQueryClient);
  const mode = useThemeMode();
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={client}>
        <AuthProvider>
          <StatusBar style={mode === "dark" ? "light" : "dark"} />
          <RootNavigator />
          <ToastHost />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
