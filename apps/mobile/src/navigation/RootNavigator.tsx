import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React, { useMemo, useRef } from "react";
import { useAuth } from "../hooks/useAuth";
import { RealtimeProvider } from "../hooks/useRealtime";
import { AuthScreen } from "../screens/auth/AuthScreen";
import { ForgotPasswordScreen } from "../screens/auth/ForgotPasswordScreen";
import { ChatScreen } from "../screens/messages/ChatScreen";
import { NotificationsScreen } from "../screens/messages/NotificationsScreen";
import { MatchScreen } from "../screens/discover/MatchScreen";
import { ProfileDetailScreen } from "../screens/discover/ProfileDetailScreen";
import { OnboardingScreen } from "../screens/onboarding/OnboardingScreen";
import { BlockedUsersScreen } from "../screens/profile/BlockedUsersScreen";
import { ChangePasswordScreen } from "../screens/profile/ChangePasswordScreen";
import { DeleteAccountScreen } from "../screens/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../screens/profile/EditProfileScreen";
import { PhotosScreen } from "../screens/profile/PhotosScreen";
import { PreferencesScreen } from "../screens/profile/PreferencesScreen";
import { PrivacyScreen } from "../screens/profile/PrivacyScreen";
import { ReportScreen } from "../screens/safety/ReportScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { useTheme } from "../theme";
import { MainTabs } from "./MainTabs";
import type { AuthStackParamList, RootStackParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<RootStackParamList>();

function useNavigationTheme(): Theme {
  const p = useTheme();
  const base = p.scheme === "dark" ? DarkTheme : DefaultTheme;
  return useMemo(
    () => ({
      ...base,
      colors: {
        ...base.colors,
        primary: p.primary,
        background: p.background,
        card: p.background,
        text: p.text,
        border: p.border,
        notification: p.primary
      }
    }),
    [base, p]
  );
}

function AuthFlow() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Auth" component={AuthScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

const headerOptions = { headerShadowVisible: false, headerBackTitle: "Retour" } as const;

function MainFlow() {
  return (
    <RealtimeProvider>
      <AppStack.Navigator screenOptions={headerOptions}>
        <AppStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
        <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "Profil" }} />
        <AppStack.Screen name="Chat" component={ChatScreen} options={({ route }) => ({ title: route.params.name })} />
        <AppStack.Screen name="Notifications" component={NotificationsScreen} options={{ title: "Notifications" }} />
        <AppStack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier le profil" }} />
        <AppStack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
        <AppStack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Préférences de rencontre" }} />
        <AppStack.Screen name="Privacy" component={PrivacyScreen} options={{ title: "Confidentialité" }} />
        <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} options={{ title: "Personnes bloquées" }} />
        <AppStack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ title: "Mot de passe" }} />
        <AppStack.Screen name="DeleteAccount" component={DeleteAccountScreen} options={{ title: "Supprimer mon compte" }} />
        <AppStack.Screen name="Report" component={ReportScreen} options={{ title: "Signaler", presentation: "modal" }} />
        <AppStack.Screen
          name="Match"
          component={MatchScreen}
          options={{ headerShown: false, presentation: "transparentModal", animation: "fade" }}
        />
      </AppStack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status, user, retryBoot } = useAuth();
  const theme = useNavigationTheme();

  // Decided once per sign-in so finishing the last onboarding step does not unmount the wizard mid-flight.
  const onboardingFor = useRef<{ userId: string; needed: boolean } | null>(null);
  const [, force] = React.useReducer((n: number) => n + 1, 0);
  if (user && onboardingFor.current?.userId !== user.id) {
    onboardingFor.current = { userId: user.id, needed: !user.hasProfile || user.photoCount < 1 };
  }
  if (!user) onboardingFor.current = null;

  let content: React.ReactNode;
  if (status === "booting") {
    content = (
      <SafeAreaLayout>
        <LoadingView label="Ouverture d'Alma…" />
      </SafeAreaLayout>
    );
  } else if (status === "unreachable") {
    content = (
      <SafeAreaLayout>
        <ErrorView
          title="Connexion impossible"
          message="Alma ne parvient pas à joindre le serveur. Vérifiez votre connexion."
          onRetry={retryBoot}
        />
      </SafeAreaLayout>
    );
  } else if (status === "signedOut" || !user) {
    content = <AuthFlow />;
  } else if (onboardingFor.current?.needed) {
    content = (
      <OnboardingScreen
        onDone={() => {
          if (onboardingFor.current) onboardingFor.current.needed = false;
          force();
        }}
      />
    );
  } else {
    content = <MainFlow />;
  }

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
