import { DefaultTheme, DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import React, { useMemo } from "react";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { useAuth } from "../hooks/useAuth";
import { useOwnProfile } from "../hooks/useProfile";
import { useOnboardingHold } from "../store/onboardingHold";
import { useColors, useThemeMode } from "../theme";
import { ErrorState, LoadingState } from "../ui";
import { AuthNavigator } from "./AuthNavigator";
import { MainNavigator } from "./MainNavigator";

function SignedIn() {
  const profile = useOwnProfile();
  const held = useOnboardingHold();

  if (profile.isPending) return <LoadingState label="Loading your profile" />;
  if (profile.isError) {
    return (
      <ErrorState error={profile.error} onRetry={() => void profile.refetch()} title="We couldn't load your profile" />
    );
  }
  const needsOnboarding = held || !profile.data || !profile.data.completed;
  if (needsOnboarding) return <OnboardingScreen key="onboarding" />;
  return <MainNavigator />;
}

export function RootNavigator() {
  const { status, bootError, retryBoot } = useAuth();
  const colors = useColors();
  const mode = useThemeMode();

  const theme = useMemo<Theme>(() => {
    const base = mode === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: colors.primary,
        background: colors.background,
        card: colors.surface,
        text: colors.text,
        border: colors.border,
        notification: colors.primary
      }
    };
  }, [colors, mode]);

  let content: React.ReactNode;
  if (status === "booting") content = <LoadingState label="Restoring your session" />;
  else if (status === "bootError")
    content = <ErrorState error={bootError} onRetry={retryBoot} title="Can't reach Lumen" />;
  else if (status === "signedOut") content = <AuthNavigator />;
  else content = <SignedIn />;

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
