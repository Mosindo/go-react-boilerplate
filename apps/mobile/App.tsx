import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { queryClient } from "./src/state/queryClient";
import { SessionProvider } from "./src/state/session";
import { ToastProvider } from "./src/ui";

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <SessionProvider>
            <StatusBar style="auto" />
            <RootNavigator />
          </SessionProvider>
        </ToastProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
