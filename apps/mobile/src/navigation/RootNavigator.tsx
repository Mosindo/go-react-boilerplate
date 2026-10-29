import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React, { useMemo } from "react";
import { MatchModal } from "../components/MatchModal";
import { ErrorView, LoadingView } from "../components/StateViews";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import ResetPasswordScreen from "../screens/auth/ResetPasswordScreen";
import SignInScreen from "../screens/auth/SignInScreen";
import SignUpScreen from "../screens/auth/SignUpScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import BlockedScreen from "../screens/main/BlockedScreen";
import ChangePasswordScreen from "../screens/main/ChangePasswordScreen";
import ChatScreen from "../screens/main/ChatScreen";
import EditProfileScreen from "../screens/main/EditProfileScreen";
import PhotosScreen from "../screens/main/PhotosScreen";
import PreferencesScreen from "../screens/main/PreferencesScreen";
import ProfileDetailScreen from "../screens/main/ProfileDetailScreen";
import SettingsScreen from "../screens/main/SettingsScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import { useAuth } from "../state/AuthProvider";
import { RealtimeProvider } from "../state/RealtimeProvider";
import { useTheme } from "../theme/ThemeProvider";
import MainTabs from "./MainTabs";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

function AuthenticatedStack() {
  return (
    <>
      <Stack.Navigator
        screenOptions={{ headerBackTitleVisible: false, headerShadowVisible: false }}
      >
        <Stack.Screen component={MainTabs} name="Main" options={{ headerShown: false }} />
        <Stack.Screen component={ChatScreen} name="Chat" />
        <Stack.Screen
          component={ProfileDetailScreen}
          name="ProfileDetail"
          options={{ title: "Profil" }}
        />
        <Stack.Screen
          component={EditProfileScreen}
          name="EditProfile"
          options={{ title: "Mes informations" }}
        />
        <Stack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
        <Stack.Screen
          component={PreferencesScreen}
          name="Preferences"
          options={{ title: "Préférences" }}
        />
        <Stack.Screen
          component={SettingsScreen}
          name="Settings"
          options={{ title: "Confidentialité et compte" }}
        />
        <Stack.Screen
          component={BlockedScreen}
          name="Blocked"
          options={{ title: "Utilisateurs bloqués" }}
        />
        <Stack.Screen
          component={ChangePasswordScreen}
          name="ChangePassword"
          options={{ title: "Mot de passe" }}
        />
      </Stack.Navigator>
      <MatchModal />
    </>
  );
}

export default function RootNavigator() {
  const palette = useTheme();
  const { state, retry } = useAuth();

  const navigationTheme = useMemo<Theme>(() => {
    const base = palette.isDark ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: palette.primary,
        background: palette.background,
        card: palette.background,
        text: palette.text,
        border: palette.border,
        notification: palette.primary
      }
    };
  }, [palette]);

  let content: React.ReactNode;
  if (state.status === "loading") {
    content = <LoadingView label="Un instant…" />;
  } else if (state.status === "offline") {
    content = <ErrorView message={state.message} onRetry={retry} />;
  } else if (state.status === "signedOut") {
    content = (
      <Stack.Navigator
        screenOptions={{
          headerBackTitleVisible: false,
          headerShadowVisible: false,
          headerTitle: ""
        }}
      >
        <Stack.Screen component={WelcomeScreen} name="Welcome" options={{ headerShown: false }} />
        <Stack.Screen component={SignInScreen} name="SignIn" />
        <Stack.Screen component={SignUpScreen} name="SignUp" />
        <Stack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        <Stack.Screen component={ResetPasswordScreen} name="ResetPassword" />
      </Stack.Navigator>
    );
  } else if (!state.me.onboardingComplete) {
    content = (
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        <Stack.Screen component={OnboardingScreen} name="Onboarding" />
      </Stack.Navigator>
    );
  } else {
    content = (
      <RealtimeProvider>
        <AuthenticatedStack />
      </RealtimeProvider>
    );
  }

  return <NavigationContainer theme={navigationTheme}>{content}</NavigationContainer>;
}
