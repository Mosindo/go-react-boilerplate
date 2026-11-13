import { StatusBar } from "expo-status-bar";
import React from "react";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/state/realtime";
import { SessionProvider } from "./src/state/session";
import { ThemeProvider, useTheme } from "./src/theme/ThemeProvider";
import { ToastProvider } from "./src/ui";

function ThemedStatusBar() {
  const { dark } = useTheme();
  return <StatusBar style={dark ? "light" : "dark"} />;
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <ToastProvider>
          <SessionProvider>
            <RealtimeProvider>
              <ThemedStatusBar />
              <RootNavigator />
            </RealtimeProvider>
          </SessionProvider>
        </ToastProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
