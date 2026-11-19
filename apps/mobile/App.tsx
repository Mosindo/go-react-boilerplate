import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { MatchCelebrationProvider } from "./src/state/matchCelebration";
import { queryClient } from "./src/state/queryClient";
import { SessionProvider } from "./src/state/session";
import { ThemeProvider, useTheme } from "./src/theme/ThemeProvider";
import { FeedbackProvider } from "./src/ui";

function ThemedStatusBar() {
  const { scheme } = useTheme();
  return <StatusBar style={scheme === "dark" ? "light" : "dark"} />;
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <ThemedStatusBar />
        <QueryClientProvider client={queryClient}>
          <FeedbackProvider>
            <SessionProvider>
              <MatchCelebrationProvider>
                <RootNavigator />
              </MatchCelebrationProvider>
            </SessionProvider>
          </FeedbackProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
