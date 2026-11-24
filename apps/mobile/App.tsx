import React from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthProvider";
import { ToastHost } from "./src/components/Toast";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { queryClient } from "./src/queryClient";

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <StatusBar style="light" />
          <RootNavigator />
          <ToastHost />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
