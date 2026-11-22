import React, { useRef } from "react";
import { DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth, useCurrentUser } from "../hooks/useAuth";
import { useMyProfile } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { ActiveChatContext } from "../hooks/activeChat";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import SignInScreen from "../screens/auth/SignInScreen";
import SignUpScreen from "../screens/auth/SignUpScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import ChatThreadScreen from "../screens/chat/ChatThreadScreen";
import NotificationsScreen from "../screens/chat/NotificationsScreen";
import ProfileDetailScreen from "../screens/discover/ProfileDetailScreen";
import OnboardingScreen, { needsOnboarding } from "../screens/onboarding/OnboardingScreen";
import BlockedUsersScreen from "../screens/profile/BlockedUsersScreen";
import ChangePasswordScreen from "../screens/profile/ChangePasswordScreen";
import DeleteAccountScreen from "../screens/profile/DeleteAccountScreen";
import EditProfileScreen from "../screens/profile/EditProfileScreen";
import PreferencesScreen from "../screens/profile/PreferencesScreen";
import SettingsScreen from "../screens/profile/SettingsScreen";
import { ErrorState, LoadingState, Screen, colors } from "../ui";
import { errorMessage } from "../api/client";
import { MainTabs } from "./MainTabs";
import type { AuthStackParamList, MainStackParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();

const theme: Theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    background: colors.background,
    card: colors.surface,
    border: colors.border,
    primary: colors.accent,
    text: colors.text,
    notification: colors.accent
  }
};

const screenOptions = { headerShown: false, contentStyle: { backgroundColor: colors.background } } as const;

function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={screenOptions}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" />
      <AuthStack.Screen component={SignInScreen} name="SignIn" />
      <AuthStack.Screen component={SignUpScreen} name="SignUp" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
    </AuthStack.Navigator>
  );
}

function SignedInNavigator() {
  const user = useCurrentUser();
  const activeChat = useRef<string | null>(null);
  useRealtime(user.id, activeChat);
  const profile = useMyProfile();

  if (profile.isPending) {
    return (
      <Screen>
        <LoadingState label="Préparation de votre espace…" />
      </Screen>
    );
  }
  if (profile.isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profile.error)} onAction={() => void profile.refetch()} />
      </Screen>
    );
  }

  return (
    <ActiveChatContext.Provider value={activeChat}>
      <MainStack.Navigator initialRouteName={needsOnboarding(profile.data) ? "Onboarding" : "Tabs"} screenOptions={screenOptions}>
        <MainStack.Screen component={MainTabs} name="Tabs" />
        <MainStack.Screen component={OnboardingScreen} name="Onboarding" options={{ gestureEnabled: false }} />
        <MainStack.Screen component={ChatThreadScreen} name="ChatThread" />
        <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ presentation: "modal" }} />
        <MainStack.Screen component={EditProfileScreen} name="EditProfile" />
        <MainStack.Screen component={PreferencesScreen} name="Preferences" />
        <MainStack.Screen component={SettingsScreen} name="Settings" />
        <MainStack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
        <MainStack.Screen component={ChangePasswordScreen} name="ChangePassword" />
        <MainStack.Screen component={DeleteAccountScreen} name="DeleteAccount" />
        <MainStack.Screen component={NotificationsScreen} name="Notifications" />
      </MainStack.Navigator>
    </ActiveChatContext.Provider>
  );
}

export function RootNavigator() {
  const { status, retryBoot } = useAuth();

  if (status === "booting") {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (status === "offline") {
    return (
      <Screen>
        <ErrorState message="Impossible de joindre Lueur. Vérifiez votre connexion." onAction={retryBoot} title="Hors connexion" />
      </Screen>
    );
  }
  return <NavigationContainer theme={theme}>{status === "signedIn" ? <SignedInNavigator /> : <AuthNavigator />}</NavigationContainer>;
}
