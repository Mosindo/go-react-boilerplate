import React, { useEffect, useMemo, useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { useAuth } from "../features/auth/AuthProvider";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { ConversationScreen } from "../features/chat/ConversationScreen";
import { ProfileDetailScreen } from "../features/discovery/ProfileDetailScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { EditPhotosScreen } from "../features/profile/EditPhotosScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { useMyProfile } from "../features/profile/hooks";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { SettingsScreen } from "../features/profile/SettingsScreen";
import { BlockedUsersScreen } from "../features/safety/BlockedUsersScreen";
import { useTheme } from "../theme";
import { ErrorState, LoadingState, Screen } from "../ui";
import { navigationRef } from "./navigationRef";
import { OnboardingGateContext } from "./onboardingGate";
import { Tabs } from "./Tabs";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

export function RootNavigator() {
  const theme = useTheme();
  const { status, user, retryBoot } = useAuth();
  const profile = useMyProfile(status === "signedIn");
  // Onboarding is decided once per session so that completing a step (e.g.
  // adding the first photo) doesn't yank the member out of the flow.
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    setOnboarding(null);
  }, [user?.id]);
  useEffect(() => {
    if (onboarding === null && profile.data) {
      setOnboarding(!profile.data.isComplete);
    }
  }, [onboarding, profile.data]);

  const navTheme = useMemo(() => {
    const base = theme.scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        background: theme.colors.background,
        card: theme.colors.surface,
        border: theme.colors.border,
        primary: theme.colors.primary,
        text: theme.colors.text,
        notification: theme.colors.primary
      }
    };
  }, [theme]);
  const gate = useMemo(() => ({ finish: () => setOnboarding(false) }), []);

  if (status === "booting" || (status === "signedIn" && (profile.isLoading || (profile.data && onboarding === null)))) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (status === "offline" || (status === "signedIn" && profile.isError)) {
    return (
      <Screen>
        <ErrorState
          message={status === "offline" ? "Impossible de joindre Lueur. Vérifiez votre connexion." : errorMessage(profile.error)}
          onRetry={() => (status === "offline" ? retryBoot() : void profile.refetch())}
        />
      </Screen>
    );
  }

  return (
    <OnboardingGateContext.Provider value={gate}>
      <NavigationContainer ref={navigationRef} theme={navTheme} documentTitle={{ formatter: () => "Lueur" }}>
        <Stack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: theme.colors.background } }}>
          {status !== "signedIn" ? (
            <>
              <Stack.Screen name="Welcome" component={WelcomeScreen} />
              <Stack.Screen name="Login" component={LoginScreen} />
              <Stack.Screen name="Register" component={RegisterScreen} />
              <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
            </>
          ) : onboarding ? (
            <Stack.Screen name="Onboarding" component={OnboardingScreen} />
          ) : (
            <>
              <Stack.Screen name="Tabs" component={Tabs} />
              <Stack.Screen name="Conversation" component={ConversationScreen} />
              <Stack.Screen
                name="ProfileDetail"
                component={ProfileDetailScreen}
                options={{ presentation: "modal", animation: "slide_from_bottom" }}
              />
              <Stack.Screen name="EditProfile" component={EditProfileScreen} />
              <Stack.Screen name="EditPhotos" component={EditPhotosScreen} />
              <Stack.Screen name="Preferences" component={PreferencesScreen} />
              <Stack.Screen name="Settings" component={SettingsScreen} />
              <Stack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
            </>
          )}
        </Stack.Navigator>
      </NavigationContainer>
    </OnboardingGateContext.Provider>
  );
}
