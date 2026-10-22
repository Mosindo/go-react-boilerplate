import React, { useEffect } from "react";
import { StyleSheet, View } from "react-native";
import { NavigationContainer, createNavigationContainerRef } from "@react-navigation/native";
import { StatusBar } from "expo-status-bar";
import { MatchOverlay } from "../components/MatchOverlay";
import { useAuth } from "../hooks/useAuth";
import { RealtimeProvider, useRealtime } from "../hooks/useRealtime";
import { AuthNavigator } from "../navigation/AuthNavigator";
import { MainNavigator } from "../navigation/MainNavigator";
import type { RootStackParamList } from "../navigation/types";
import { clearGlobalError, ErrorView, LoadingView, useGlobalFeedback } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { colors, spacing } from "../shared/ui";
import OnboardingScreen from "./OnboardingScreen";

const navigationRef = createNavigationContainerRef<RootStackParamList>();

function ErrorBanner() {
  const feedback = useGlobalFeedback();
  useEffect(() => {
    if (!feedback.error) {
      return undefined;
    }
    const id = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(id);
  }, [feedback.error]);

  return feedback.error ? (
    <View pointerEvents="box-none" style={styles.bannerWrap}>
      <ErrorView actionLabel="Fermer" compact message={feedback.error} onAction={clearGlobalError} title="Un souci" />
    </View>
  ) : null;
}

function MatchHost() {
  const { pendingMatch, dismissMatch } = useRealtime();
  return (
    <MatchOverlay
      match={pendingMatch}
      onChat={(match) => {
        dismissMatch();
        if (navigationRef.isReady()) {
          navigationRef.navigate("Chat", { conversationId: match.conversationId, peerId: match.user.userId, name: match.user.firstName });
        }
      }}
      onDismiss={dismissMatch}
    />
  );
}

/** Chooses what to show: loading, sign-in, onboarding or the app. */
export default function Root() {
  const { isAuthenticated, isBooting, user } = useAuth();

  if (isBooting) {
    return (
      <SafeAreaLayout>
        <StatusBar style="dark" />
        <LoadingView fullScreen label="Chargement…" />
      </SafeAreaLayout>
    );
  }

  if (!isAuthenticated || !user) {
    return (
      <NavigationContainer>
        <StatusBar style="dark" />
        <AuthNavigator />
        <ErrorBanner />
      </NavigationContainer>
    );
  }

  if (!user.profileComplete) {
    return (
      <>
        <StatusBar style="dark" />
        <OnboardingScreen />
        <ErrorBanner />
      </>
    );
  }

  return (
    <RealtimeProvider>
      <NavigationContainer ref={navigationRef}>
        <StatusBar style="dark" />
        <MainNavigator />
        <MatchHost />
        <ErrorBanner />
      </NavigationContainer>
    </RealtimeProvider>
  );
}

const styles = StyleSheet.create({
  bannerWrap: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    bottom: spacing.xxxl + spacing.xl,
    backgroundColor: colors.background,
    borderRadius: 20
  }
});
