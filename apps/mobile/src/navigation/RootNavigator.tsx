import React, { useCallback, useState } from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { BrandMark, ErrorState, Screen } from "../shared/ui";
import { spacing, useTheme } from "../theme";
import { useAuth } from "../features/auth/AuthProvider";
import { OnboardingProvider } from "../features/onboarding/OnboardingContext";
import { MatchCelebration } from "../features/matches/MatchCelebration";
import { useProfileState } from "../features/profile/hooks";
import { useConversations, useMatches } from "../features/matches/hooks";
import { useRealtime } from "../realtime/RealtimeProvider";
import type { ProfileState } from "../api";
import { AuthNavigator, MainNavigator, OnboardingNavigator } from "./stacks";

function Splash() {
  const { colors } = useTheme();
  return (
    <Screen edges={["top", "bottom"]} testID="splash">
      <View style={styles.splash}>
        <BrandMark size={56} />
        <ActivityIndicator color={colors.primary} />
      </View>
    </Screen>
  );
}

function MainApp() {
  useRealtime();
  // Keep the lists warm: the chat menu and the match strip resolve ids from these caches.
  useMatches();
  useConversations();
  return (
    <>
      <MainNavigator />
      <MatchCelebration />
    </>
  );
}

function SignedInGate({ initial }: { initial: ProfileState }) {
  // Decided once: onboarding must stay open after the last required step (photos) completes the profile.
  const [onboarding, setOnboarding] = useState(!initial.complete);
  const finish = useCallback(() => setOnboarding(false), []);

  if (onboarding) {
    return (
      <OnboardingProvider initial={initial} onFinish={finish}>
        <OnboardingNavigator />
      </OnboardingProvider>
    );
  }
  return <MainApp />;
}

function SignedIn() {
  const profile = useProfileState();
  if (profile.isError) {
    return (
      <Screen edges={["top", "bottom"]}>
        <ErrorState onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  if (!profile.data) {
    return <Splash />;
  }
  return <SignedInGate initial={profile.data} />;
}

export function RootNavigator() {
  const { status, bootFailed, retryBoot } = useAuth();

  if (bootFailed) {
    return (
      <Screen edges={["top", "bottom"]}>
        <ErrorState message="Impossible de joindre le serveur." onRetry={retryBoot} />
      </Screen>
    );
  }
  if (status === "booting") {
    return <Splash />;
  }
  return status === "signedIn" ? <SignedIn /> : <AuthNavigator />;
}

const styles = StyleSheet.create({
  splash: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.xl }
});
