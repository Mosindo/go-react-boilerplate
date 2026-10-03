import React from "react";
import { StatusBar } from "expo-status-bar";
import { StyleSheet, View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import { RealtimeProvider } from "./src/hooks/useRealtime";
import { AppNavigator } from "./src/navigation/AppNavigator";
import AuthScreen from "./src/screens/AuthScreen";
import { SafeAreaLayout } from "./src/shared/layout";
import { LoadingView, ToastHost, clearGlobalError, showToast, useGlobalFeedback } from "./src/shared/feedback";
import { ErrorView } from "./src/shared/feedback";
import { DialogHost } from "./src/shared/dialog";
import { colors, spacing } from "./src/shared/ui";

function AppShell() {
  const { isAuthenticated, isBooting } = useAuth();
  const feedback = useGlobalFeedback();

  if (isBooting) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Opening Amora…" />
      </SafeAreaLayout>
    );
  }

  if (!isAuthenticated) {
    return <AuthScreen />;
  }

  return (
    <RealtimeProvider onToast={(message) => showToast(message)}>
      <AppNavigator />
      {feedback.error ? (
        <View pointerEvents="box-none" style={styles.bannerWrap}>
          <ErrorView actionLabel="Dismiss" compact message={feedback.error} onAction={clearGlobalError} title="Request issue" />
        </View>
      ) : null}
      {feedback.loadingCount > 0 ? (
        <View style={styles.overlay}>
          <LoadingView label={feedback.loadingLabel ?? "Working..."} style={styles.overlayCard} />
        </View>
      ) : null}
    </RealtimeProvider>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <StatusBar style="dark" />
        <AppShell />
        <ToastHost />
        <DialogHost />
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  bannerWrap: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.xxl },
  overlay: { ...StyleSheet.absoluteFillObject, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", paddingHorizontal: spacing.xxl },
  overlayCard: { width: "100%", maxWidth: 320 }
});
