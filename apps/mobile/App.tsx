import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { ApiError } from "./src/api/client";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { Toaster } from "./src/shared/feedback";
import { ThemeProvider, useTheme } from "./src/theme";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Do not hammer the API on client errors (401/404/409…); retry only transient failures.
        retry: (failures, error) => !(error instanceof ApiError && error.status < 500) && failures < 2,
        refetchOnWindowFocus: false
      }
    }
  });
}

function ThemedStatusBar() {
  const { isDark } = useTheme();
  return <StatusBar style={isDark ? "light" : "dark"} />;
}

export default function App() {
  const [queryClient] = useState(createQueryClient);
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <ThemedStatusBar />
            <RootNavigator />
            <Toaster />
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
