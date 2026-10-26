import React, { useState } from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthContext";
import { FeedbackProvider } from "./src/components/Feedback";
import { AppNavigator } from "./src/navigation";
import { RealtimeProvider } from "./src/realtime/RealtimeProvider";
import { ThemeProvider, useTheme } from "./src/theme";

function ThemedStatusBar() {
  const { scheme } = useTheme();
  return <StatusBar style={scheme === "dark" ? "light" : "dark"} />;
}

export default function App() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false },
          mutations: { retry: 0 }
        }
      })
  );

  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <FeedbackProvider>
            <AuthProvider>
              <RealtimeProvider>
                <ThemedStatusBar />
                <AppNavigator />
              </RealtimeProvider>
            </AuthProvider>
          </FeedbackProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
