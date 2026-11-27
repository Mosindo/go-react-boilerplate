import React, { useEffect, useMemo, useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { useMatches, useMyProfile, useUnreadNotifications } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { useRealtimeSync } from "../hooks/useRealtime";
import { useTheme } from "../theme/ThemeProvider";
import { ErrorState, Icon, Loading, Screen, type IconName } from "../ui";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discover/ProfileDetailScreen";
import { ChatScreen } from "../features/matches/ChatScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { DeleteAccountScreen } from "../features/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { PrivacyScreen } from "../features/profile/PrivacyScreen";
import { BlockedUsersScreen } from "../features/safety/BlockedUsersScreen";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const TAB_ICONS: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

const TAB_LABELS: Record<keyof TabParamList, string> = {
  Discover: "Découvrir",
  Matches: "Matchs",
  Notifications: "Activité",
  Me: "Moi"
};

function MainTabs() {
  const { colors } = useTheme();
  const matches = useMatches();
  const unreadNotifications = useUnreadNotifications();
  const unreadMessages = useMemo(
    () => (matches.data?.pages ?? []).flatMap((p) => p.matches).reduce((sum, m) => sum + m.unreadCount, 0),
    [matches.data]
  );

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabel: TAB_LABELS[route.name],
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarButtonTestID: `tab-${route.name}`,
        tabBarIcon: ({ focused, color, size }) => <Icon name={TAB_ICONS[route.name][focused ? 0 : 1]} size={size} color={color} />
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} />
      <Tabs.Screen name="Matches" component={MatchesScreen} options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined }} />
      <Tabs.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{ tabBarBadge: unreadNotifications.data ? unreadNotifications.data : undefined }}
      />
      <Tabs.Screen name="Me" component={MeScreen} />
    </Tabs.Navigator>
  );
}

function SignedInApp() {
  const { user } = useAuth();
  useRealtimeSync(user?.id ?? "");
  const profile = useMyProfile();
  // Decided once per session so the wizard is not interrupted the moment the
  // profile technically becomes complete (e.g. right after the first photo).
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    if (profile.data && onboarding === null) {
      setOnboarding(!profile.data.isComplete);
    }
  }, [profile.data, onboarding]);

  if (profile.isLoading || (profile.data && onboarding === null)) {
    return <Loading label="Chargement de ton profil…" />;
  }
  if (profile.error || !profile.data) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  if (onboarding) {
    return <OnboardingScreen onDone={() => setOnboarding(false)} />;
  }

  return (
    <AppStack.Navigator screenOptions={{ headerShown: false }}>
      <AppStack.Screen name="Tabs" component={MainTabs} />
      <AppStack.Screen name="Chat" component={ChatScreen} />
      <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ animation: "slide_from_bottom" }} />
      <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
      <AppStack.Screen name="Preferences" component={PreferencesScreen} />
      <AppStack.Screen name="Privacy" component={PrivacyScreen} />
      <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
      <AppStack.Screen name="DeleteAccount" component={DeleteAccountScreen} />
    </AppStack.Navigator>
  );
}

function SignedOutApp() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, retryBoot } = useAuth();
  const { colors, scheme } = useTheme();
  const navTheme = useMemo<Theme>(() => {
    const base = scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: { ...base.colors, background: colors.background, card: colors.surface, border: colors.border, primary: colors.primary, text: colors.text, notification: colors.primary }
    };
  }, [colors, scheme]);

  let content: React.ReactNode;
  if (status === "booting") {
    content = <Loading />;
  } else if (status === "offline") {
    content = (
      <Screen>
        <ErrorState message="Impossible de joindre le serveur. Vérifie ta connexion." onRetry={retryBoot} />
      </Screen>
    );
  } else if (status === "signedIn") {
    content = <SignedInApp />;
  } else {
    content = <SignedOutApp />;
  }

  return <NavigationContainer theme={navTheme}>{content}</NavigationContainer>;
}
