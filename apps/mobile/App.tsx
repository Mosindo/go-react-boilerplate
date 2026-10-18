import React, { useEffect } from "react";
import { StatusBar } from "expo-status-bar";
import { StyleSheet, View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/hooks/useAuth";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { clearGlobalError, ErrorView, ToastHost, useGlobalFeedback } from "./src/shared/feedback";
import { spacing } from "./src/shared/ui";

function ErrorBanner() {
  const { error } = useGlobalFeedback();

  useEffect(() => {
    if (!error) {
      return;
    }
    const timeoutId = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(timeoutId);
  }, [error]);

  if (!error) {
    return null;
  }
  return (
    <View pointerEvents="box-none" style={styles.bannerWrap}>
      <ErrorView actionLabel="Fermer" compact message={error} onAction={clearGlobalError} title="Problème de connexion" />
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <StatusBar style="dark" />
        <RootNavigator />
        <ToastHost />
        <ErrorBanner />
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  bannerWrap: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: 96 }
});
