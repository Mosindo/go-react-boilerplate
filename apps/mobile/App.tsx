import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthContext";
import { MatchCelebrationProvider } from "./src/features/discover/matchCelebration";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeProvider";
import { ThemeProvider, useTheme } from "./src/theme/theme";
import { ToastProvider } from "./src/ui/Toast";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

function Shell() {
  const { dark } = useTheme();
  return (
    <>
      <StatusBar style={dark ? "light" : "dark"} />
      <RootNavigator />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <ToastProvider>
            <AuthProvider>
              <RealtimeProvider>
                <MatchCelebrationProvider>
                  <Shell />
                </MatchCelebrationProvider>
              </RealtimeProvider>
            </AuthProvider>
          </ToastProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
