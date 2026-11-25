import React from "react";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { SessionProvider } from "./src/hooks/useSession";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { ToastHost } from "./src/ui";

export default function App() {
  return (
    <SafeAreaProvider>
      <SessionProvider>
        <StatusBar style="light" />
        <RootNavigator />
        <ToastHost />
      </SessionProvider>
    </SafeAreaProvider>
  );
}
