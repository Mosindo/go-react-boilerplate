import React from "react";
import { StatusBar } from "expo-status-bar";
import { StyleSheet, View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { FeedbackHost } from "./src/shared/feedback";
import { colors } from "./src/shared/ui";

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <View style={styles.root}>
          <StatusBar style="light" />
          <RootNavigator />
          <FeedbackHost />
        </View>
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: colors.background
  }
});
