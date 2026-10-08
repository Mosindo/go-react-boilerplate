import React, { useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavigationTheme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { MatchModal } from "../components/MatchModal";
import { useMe } from "../hooks/data";
import { useAuth } from "../hooks/useAuth";
import AuthScreen from "../screens/AuthScreen";
import BlockedScreen from "../screens/BlockedScreen";
import ChangePasswordScreen from "../screens/ChangePasswordScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ForgotPasswordScreen from "../screens/ForgotPasswordScreen";
import HomeScreen from "../screens/HomeScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ReportScreen from "../screens/ReportScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button } from "../shared/ui";
import { useTheme } from "../theme";
import { navigationRef } from "./ref";
import type { AuthStackParamList, MainStackParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();

function useNavigationTheme(): NavigationTheme {
  const { colors, scheme } = useTheme();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  return {
    ...base,
    colors: {
      ...base.colors,
      primary: colors.primary,
      background: colors.background,
      card: colors.background,
      text: colors.text,
      border: colors.border,
      notification: colors.primary
    }
  };
}

function MainNavigator() {
  const { colors } = useTheme();
  return (
    <>
      <MainStack.Navigator
        screenOptions={{
          headerShadowVisible: false,
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
          headerTitleStyle: { fontWeight: "700" },
          headerBackTitle: "Retour",
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <MainStack.Screen component={HomeScreen} name="Tabs" options={{ headerShown: false }} />
        <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "" }} />
        <MainStack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
        <MainStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
        <MainStack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Préférences" }} />
        <MainStack.Screen component={SettingsScreen} name="Settings" options={{ title: "Réglages" }} />
        <MainStack.Screen component={NotificationsScreen} name="Notifications" options={{ title: "Notifications" }} />
        <MainStack.Screen component={BlockedScreen} name="Blocked" options={{ title: "Personnes bloquées" }} />
        <MainStack.Screen component={ChangePasswordScreen} name="ChangePassword" options={{ title: "Mot de passe" }} />
        <MainStack.Screen component={ReportScreen} name="Report" options={{ title: "Signaler" }} />
      </MainStack.Navigator>
      <MatchModal />
    </>
  );
}

function SignedIn() {
  const { signOut } = useAuth();
  const { data: me, isLoading, isError, refetch } = useMe();
  // Once onboarding has been shown, stay on it until the user taps the final button (adding the first photo
  // already makes the account "complete", but they should not be thrown into the app mid-flow).
  const [inOnboarding, setInOnboarding] = useState<boolean | null>(null);
  if (inOnboarding === null && me) setInOnboarding(!me.onboarding.complete);

  if (isLoading) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement…" />
      </SafeAreaLayout>
    );
  }
  if (isError || !me) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger votre compte. Vérifiez votre connexion." onAction={() => void refetch()} />
        <Button onPress={() => void signOut()} variant="ghost">
          Me déconnecter
        </Button>
      </SafeAreaLayout>
    );
  }
  if (inOnboarding || !me.onboarding.complete) return <OnboardingScreen me={me} onFinish={() => setInOnboarding(false)} />;
  return <MainNavigator />;
}

/** Chooses between booting, signed-out, onboarding and the main app from auth + account state. */
export function RootNavigator() {
  const { status } = useAuth();
  const { colors } = useTheme();
  const theme = useNavigationTheme();

  if (status === "booting") {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen />
      </SafeAreaLayout>
    );
  }

  return (
    <NavigationContainer ref={navigationRef} theme={theme}>
      {status === "signedOut" ? (
        <AuthStack.Navigator
          screenOptions={{
            headerShadowVisible: false,
            headerStyle: { backgroundColor: colors.background },
            headerTintColor: colors.text,
            contentStyle: { backgroundColor: colors.background }
          }}
        >
          <AuthStack.Screen component={AuthScreen} name="Auth" options={{ headerShown: false }} />
          <AuthStack.Screen component={ForgotPasswordScreen} name="Forgot" options={{ title: "" }} />
        </AuthStack.Navigator>
      ) : (
        <SignedIn />
      )}
    </NavigationContainer>
  );
}
