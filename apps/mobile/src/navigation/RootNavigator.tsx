import React from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useNotifications, useOwnProfile, useUnreadChats } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import ChatScreen from "../screens/matches/ChatScreen";
import MatchesScreen from "../screens/matches/MatchesScreen";
import {
  AccountScreen,
  BlockedScreen,
  EditInterestsScreen,
  EditPhotosScreen,
  EditPreferencesScreen,
  EditProfileScreen,
  PrivacyScreen
} from "../screens/me/EditScreens";
import MeScreen from "../screens/me/MeScreen";
import NotificationsScreen from "../screens/notifications/NotificationsScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import ProfileViewScreen from "../screens/profile/ProfileViewScreen";
import { useTheme } from "../theme";
import { ErrorState, Icon, Loading, Screen, type IconName } from "../ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const tabIcons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const unreadChats = useUnreadChats();
  const notifications = useNotifications();
  const badge = (n?: number) => (n && n > 0 ? (n > 99 ? "99+" : n) : undefined);
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarIcon: ({ focused, color, size }) => <Icon name={tabIcons[route.name][focused ? 0 : 1]} size={size} color={color} />
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir" }} />
      <Tabs.Screen name="Matches" component={MatchesScreen} options={{ title: "Matchs", tabBarBadge: badge(unreadChats.data) }} />
      <Tabs.Screen name="Notifications" component={NotificationsScreen} options={{ title: "Activité", tabBarBadge: badge(notifications.data?.unreadCount) }} />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil" }} />
    </Tabs.Navigator>
  );
}

function SignedInApp({ userId }: { userId: string }) {
  const { tokenVersion } = useAuth();
  const profile = useOwnProfile();
  useRealtime(userId, tokenVersion);

  if (profile.isLoading) return <Loading label="Chargement de votre profil…" />;
  if (profile.error) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  if (!profile.data || !profile.data.isComplete) {
    return <OnboardingScreen profile={profile.data ?? null} />;
  }
  return (
    <AppStack.Navigator screenOptions={{ headerShown: false }}>
      <AppStack.Screen name="Tabs" component={MainTabs} />
      <AppStack.Screen name="Chat" component={ChatScreen} />
      <AppStack.Screen name="ProfileView" component={ProfileViewScreen} options={{ presentation: "modal" }} />
      <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
      <AppStack.Screen name="EditPhotos" component={EditPhotosScreen} />
      <AppStack.Screen name="EditPreferences" component={EditPreferencesScreen} />
      <AppStack.Screen name="EditInterests" component={EditInterestsScreen} />
      <AppStack.Screen name="Privacy" component={PrivacyScreen} />
      <AppStack.Screen name="Blocked" component={BlockedScreen} />
      <AppStack.Screen name="Account" component={AccountScreen} />
    </AppStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, user, retryBoot } = useAuth();
  const { colors, isDark } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.primary,
      text: colors.text,
      notification: colors.primary
    }
  };

  let content: React.ReactNode;
  if (status === "booting") {
    content = <Loading />;
  } else if (status === "bootError") {
    content = (
      <Screen>
        <ErrorState message="Impossible de joindre le serveur. Vérifiez votre connexion." onRetry={retryBoot} />
      </Screen>
    );
  } else if (status === "signedIn" && user) {
    content = <SignedInApp key={user.id} userId={user.id} />;
  } else {
    content = (
      <AuthStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
        <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
        <AuthStack.Screen name="Login" component={LoginScreen} />
        <AuthStack.Screen name="Register" component={RegisterScreen} />
        <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      </AuthStack.Navigator>
    );
  }
  return <NavigationContainer theme={navTheme}>{content}</NavigationContainer>;
}
