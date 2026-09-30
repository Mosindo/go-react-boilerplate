import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { ThemeProvider, useTheme } from "./src/design/ThemeProvider";
import { queryClient } from "./src/lib/queryClient";
import { SessionProvider } from "./src/lib/session/SessionProvider";
import { ToastHost } from "./src/lib/toast";
import { RootNavigator } from "./src/navigation/RootNavigator";

function ThemedStatusBar() {
  const { dark } = useTheme();
  return <StatusBar style={dark ? "light" : "dark"} />;
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <SessionProvider>
            <ThemedStatusBar />
            <RootNavigator />
            <ToastHost />
          </SessionProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
