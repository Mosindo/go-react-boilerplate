import React, { useEffect, useMemo, useState } from "react";
import { DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import type { MyProfile } from "../api/types";
import { useMatches, useMyProfile, useUnreadNotifications } from "../hooks/queries";
import { wasMatchCelebrated } from "../hooks/presence";
import { useAuth } from "../hooks/useAuth";
import { useRealtime } from "../hooks/useRealtime";
import ActivityScreen from "../screens/activity/ActivityScreen";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import ChatScreen from "../screens/chat/ChatScreen";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import MatchesScreen from "../screens/matches/MatchesScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import BlockedUsersScreen from "../screens/profile/BlockedUsersScreen";
import ChangePasswordScreen from "../screens/profile/ChangePasswordScreen";
import DeleteAccountScreen from "../screens/profile/DeleteAccountScreen";
import EditProfileScreen from "../screens/profile/EditProfileScreen";
import MeScreen from "../screens/profile/MeScreen";
import PhotosScreen from "../screens/profile/PhotosScreen";
import PreferencesScreen from "../screens/profile/PreferencesScreen";
import ProfileDetailScreen from "../screens/profile/ProfileDetailScreen";
import ReportScreen from "../screens/profile/ReportScreen";
import SettingsScreen from "../screens/profile/SettingsScreen";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Text, colors } from "../shared/ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const theme: Theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: colors.primary,
    background: colors.background,
    card: colors.backgroundElevated,
    text: colors.text,
    border: colors.border,
    notification: colors.primary
  }
};

const stackScreenOptions = {
  headerStyle: { backgroundColor: colors.background },
  headerTintColor: colors.text,
  headerShadowVisible: false,
  headerTitle: "",
  headerBackTitle: "Retour",
  contentStyle: { backgroundColor: colors.background }
} as const;

function tabIcon(glyph: string) {
  return function TabIcon({ color }: { color: string }) {
    return <Text style={{ color, fontSize: 20, lineHeight: 24 }}>{glyph}</Text>;
  };
}

function Tabs() {
  const matches = useMatches();
  const unreadNotifications = useUnreadNotifications(true);
  const unreadMessages = useMemo(
    () => matches.data?.pages.flatMap((p) => p.matches).reduce((n, m) => n + m.unreadCount, 0) ?? 0,
    [matches.data]
  );
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.backgroundElevated, borderTopColor: colors.border },
        tabBarLabelStyle: { fontSize: 11.5, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary, fontSize: 11 }
      }}
    >
      <Tab.Screen
        component={DiscoverScreen}
        name="Discover"
        options={{ title: "Découvrir", tabBarIcon: tabIcon("◈"), tabBarTestID: "tab-discover" }}
      />
      <Tab.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          title: "Matchs",
          tabBarIcon: tabIcon("♥"),
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarTestID: "tab-matches"
        }}
      />
      <Tab.Screen
        component={ActivityScreen}
        name="Activity"
        options={{
          title: "Activité",
          tabBarIcon: tabIcon("✦"),
          tabBarBadge: unreadNotifications.data ? unreadNotifications.data : undefined,
          tabBarTestID: "tab-activity"
        }}
      />
      <Tab.Screen component={MeScreen} name="Me" options={{ title: "Profil", tabBarIcon: tabIcon("◉"), tabBarTestID: "tab-me" }} />
    </Tab.Navigator>
  );
}

function needsOnboarding(profile: MyProfile): boolean {
  return profile.missing.some((m) => m !== "location");
}

function SignedIn() {
  const profile = useMyProfile();
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useRealtime({
    // The person who completed the match already sees a full-screen
    // celebration; the other one gets a toast.
    onMatch: (event) =>
      setTimeout(() => {
        if (!wasMatchCelebrated(event.data.id)) {
          showToast(`Nouveau match avec ${event.data.user.firstName} !`, "success");
        }
      }, 1500)
  });

  // Decide once per load: a completed profile never re-enters onboarding
  // because a later step (bio, location) is still pending.
  useEffect(() => {
    if (profile.data && (onboarding === null || needsOnboarding(profile.data))) {
      setOnboarding(needsOnboarding(profile.data) || (onboarding ?? false));
    }
  }, [onboarding, profile.data]);

  if (profile.isError && !profile.data) {
    return <ErrorView onRetry={() => void profile.refetch()} />;
  }
  if (!profile.data || onboarding === null) {
    return <LoadingView />;
  }
  if (onboarding) {
    return <OnboardingScreen onFinish={() => setOnboarding(false)} profile={profile.data} />;
  }
  return (
    <AppStack.Navigator screenOptions={stackScreenOptions}>
      <AppStack.Screen component={Tabs} name="Tabs" options={{ headerShown: false }} />
      <AppStack.Screen component={ChatScreen} name="Chat" options={{ headerShown: false }} />
      <AppStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ headerShown: false, presentation: "modal" }} />
      <AppStack.Screen component={EditProfileScreen} name="EditProfile" />
      <AppStack.Screen component={PhotosScreen} name="Photos" />
      <AppStack.Screen component={PreferencesScreen} name="Preferences" />
      <AppStack.Screen component={SettingsScreen} name="Settings" />
      <AppStack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
      <AppStack.Screen component={ChangePasswordScreen} name="ChangePassword" />
      <AppStack.Screen component={DeleteAccountScreen} name="DeleteAccount" />
      <AppStack.Screen component={ReportScreen} name="Report" options={{ presentation: "modal" }} />
    </AppStack.Navigator>
  );
}

function SignedOut() {
  return (
    <AuthStack.Navigator screenOptions={stackScreenOptions}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" options={{ headerShown: false }} />
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, user } = useAuth();
  return (
    <NavigationContainer documentTitle={{ formatter: () => "Hearth" }} theme={theme}>
      {status === "booting" ? <LoadingView /> : status === "signedIn" ? <SignedIn key={user?.id} /> : <SignedOut />}
    </NavigationContainer>
  );
}
