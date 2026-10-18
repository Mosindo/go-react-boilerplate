import React, { useState } from "react";
import { DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { BottomNavigation } from "../shared/layout";
import { RealtimeProvider } from "../hooks/useRealtime";
import { useMyProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import AuthScreen from "../screens/AuthScreen";
import BlockedUsersScreen from "../screens/BlockedUsersScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ForgotPasswordScreen from "../screens/ForgotPasswordScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PersonScreen from "../screens/PersonScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { colors } from "../shared/ui";
import type { AuthStackParamList, MainStackParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();

const theme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    background: colors.background,
    card: colors.backgroundElevated,
    border: colors.border,
    primary: colors.primary,
    text: colors.text,
    notification: colors.primary
  }
};

function Splash({ label }: { label: string }) {
  return (
    <SafeAreaLayout>
      <LoadingView fullScreen label={label} />
    </SafeAreaLayout>
  );
}

function MainNavigator() {
  return (
    <MainStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
      <MainStack.Screen component={BottomNavigation} name="Tabs" />
      <MainStack.Screen component={ChatScreen} name="Conversation" />
      <MainStack.Screen component={PersonScreen} name="Person" />
      <MainStack.Screen component={EditProfileScreen} name="EditProfile" />
      <MainStack.Screen component={SettingsScreen} name="Settings" />
      <MainStack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
    </MainStack.Navigator>
  );
}

/**
 * Signed in: a profile with at least one photo is required to use the app. The
 * decision is taken once per session, so onboarding stays on screen (letting
 * the user add more photos) until they choose to finish.
 */
function SignedIn({ userId }: { userId: string }) {
  const { data: profile, isLoading, isError, refetch } = useMyProfile();
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  if (isLoading) {
    return <Splash label="Chargement de votre profil…" />;
  }
  if (isError) {
    return (
      <SafeAreaLayout>
        <ErrorView actionLabel="Réessayer" message="Impossible de charger votre profil." onAction={() => void refetch()} title="Oups" />
      </SafeAreaLayout>
    );
  }
  const needsOnboarding = onboarding ?? (!profile || profile.photos.length === 0);
  if (needsOnboarding) {
    if (onboarding === null) {
      setOnboarding(true);
    }
    return <OnboardingScreen onFinish={() => setOnboarding(false)} profile={profile ?? null} />;
  }
  return (
    <RealtimeProvider userId={userId}>
      <MainNavigator />
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status, user } = useAuth();

  if (status === "booting") {
    return <Splash label="Chargement…" />;
  }

  return (
    <NavigationContainer theme={theme}>
      {status === "signedIn" && user ? (
        <SignedIn key={user.id} userId={user.id} />
      ) : (
        <AuthStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
          <AuthStack.Screen component={AuthScreen} name="Auth" />
          <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        </AuthStack.Navigator>
      )}
    </NavigationContainer>
  );
}

