import React from "react";
import { StatusBar } from "expo-status-bar";
import { NavigationContainer } from "@react-navigation/native";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/features/auth/AuthProvider";
import { navigationRef } from "./src/navigation/ref";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { queryClient } from "./src/queryClient";
import { ToastHost } from "./src/shared/feedback";
import { ThemeProvider, useTheme } from "./src/theme";

function ThemedApp() {
  const { navigationTheme, isDark } = useTheme();
  return (
    <NavigationContainer ref={navigationRef} theme={navigationTheme}>
      <StatusBar style={isDark ? "light" : "dark"} />
      <RootNavigator />
      <ToastHost />
    </NavigationContainer>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <ThemedApp />
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
