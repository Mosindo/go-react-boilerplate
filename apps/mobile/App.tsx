import React, { useEffect } from "react";
import { StyleSheet, View } from "react-native";
import { StatusBar } from "expo-status-bar";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthContext";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { clearGlobalError, ErrorView, useGlobalFeedback } from "./src/shared/feedback";
import { spacing } from "./src/shared/ui";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

function ErrorBanner() {
  const { error } = useGlobalFeedback();
  useEffect(() => {
    if (!error) return;
    const timer = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(timer);
  }, [error]);

  if (!error) return null;
  return (
    <View pointerEvents="box-none" style={styles.bannerWrap}>
      <ErrorView actionLabel="Fermer" compact message={error} onAction={clearGlobalError} title="Problème de connexion" />
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <StatusBar style="dark" />
          <RootNavigator />
          <ErrorBanner />
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  bannerWrap: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.xxl }
});
