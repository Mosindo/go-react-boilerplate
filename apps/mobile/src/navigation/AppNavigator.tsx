import React, { useEffect, useState } from "react";
import { DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { errorMessage, getOwnProfile, queryKeys } from "../api/platform";
import { BottomNavigation } from "../shared/layout";
import { ErrorView, LoadingView } from "../shared/feedback";
import { colors } from "../shared/ui";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import SettingsScreen from "../screens/SettingsScreen";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

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

/**
 * Signed-in shell. A profile with at least one photo is the entry ticket to the app;
 * until then the onboarding wizard resumes exactly where the user left off.
 */
export function AppNavigator() {
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const needsOnboarding = profile.isSuccess && (!profile.data || profile.data.photos.length === 0);
  // Once the wizard has started it stays up until the user taps "Start meeting people",
  // so adding the first photo does not yank them out of the flow.
  const [wizardActive, setWizardActive] = useState(false);
  useEffect(() => {
    if (needsOnboarding) {
      setWizardActive(true);
    }
  }, [needsOnboarding]);

  if (profile.isLoading) {
    return <LoadingView fullScreen label="Loading your profile…" />;
  }
  if (profile.isError) {
    return <ErrorView message={errorMessage(profile.error)} onAction={() => void profile.refetch()} />;
  }
  if (needsOnboarding || wizardActive) {
    return <OnboardingScreen onFinish={() => setWizardActive(false)} profile={profile.data ?? null} />;
  }

  return (
    <NavigationContainer theme={theme}>
      <Stack.Navigator screenOptions={{ headerTintColor: colors.text, headerShadowVisible: false, headerBackTitle: "Back", contentStyle: { backgroundColor: colors.background } }}>
        <Stack.Screen component={BottomNavigation} name="Tabs" options={{ headerShown: false }} />
        <Stack.Screen component={ChatScreen} name="Chat" options={({ route }) => ({ title: route.params.name })} />
        <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={({ route }) => ({ title: route.params.name })} />
        <Stack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Edit profile" }} />
        <Stack.Screen component={PhotosScreen} name="Photos" options={{ title: "Photos" }} />
        <Stack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Discovery preferences" }} />
        <Stack.Screen component={SettingsScreen} name="Settings" options={{ title: "Privacy & account" }} />
      </Stack.Navigator>
    </NavigationContainer>
  );
}
