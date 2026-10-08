import React from "react";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/hooks/useAuth";
import { RealtimeProvider } from "./src/hooks/useRealtime";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { ToastHost } from "./src/shared/feedback";
import { ThemeProvider, useTheme } from "./src/theme";

function Shell() {
  const { scheme } = useTheme();
  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      <RootNavigator />
      <ToastHost />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <AuthProvider>
          <RealtimeProvider>
            <Shell />
          </RealtimeProvider>
        </AuthProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
