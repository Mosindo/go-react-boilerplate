import React from "react";
import { StatusBar } from "expo-status-bar";
import { View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { ThemeProvider, useTheme } from "./src/theme";
import { ToastHost } from "./src/ui";

function Shell() {
  const { colors, isDark } = useTheme();
  return (
    <View style={{ flex: 1, backgroundColor: colors.background }}>
      <StatusBar style={isDark ? "light" : "dark"} />
      <RootNavigator />
      <ToastHost />
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <AuthProvider>
          <Shell />
        </AuthProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
