import React from "react";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./auth/AuthProvider";
import { FeedbackProvider } from "./components/Feedback";
import { RootNavigator } from "./navigation/RootNavigator";
import { useTheme } from "./theme/theme";

function Shell() {
  const t = useTheme();
  return (
    <>
      <StatusBar style={t.isDark ? "light" : "dark"} />
      <RootNavigator />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <FeedbackProvider>
          <Shell />
        </FeedbackProvider>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
