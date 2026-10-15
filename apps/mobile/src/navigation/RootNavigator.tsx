import React, { useEffect } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useConversations, useMyProfile, useNotifications } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { RealtimeProvider, useRealtime } from "../hooks/useRealtime";
import AuthScreen from "../screens/AuthScreen";
import BlockedScreen from "../screens/BlockedScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ForgotPasswordScreen from "../screens/ForgotPasswordScreen";
import HomeScreen from "../screens/HomeScreen";
import MatchesScreen from "../screens/MatchesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ProfileScreen from "../screens/ProfileScreen";
import ReportScreen from "../screens/ReportScreen";
import SecurityScreen from "../screens/SecurityScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { BottomNavigation, SafeAreaLayout, type TabMeta } from "../shared/layout";
import { useTheme } from "../theme";
import { navigationRef } from "./ref";
import type { RootStackParamList, TabParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const tabMeta: Record<string, TabMeta> = {
  Discover: { label: "Découvrir", icon: "flame-outline", iconActive: "flame" },
  Matches: { label: "Matchs", icon: "chatbubbles-outline", iconActive: "chatbubbles" },
  Notifications: { label: "Notifications", icon: "notifications-outline", iconActive: "notifications" },
  Me: { label: "Profil", icon: "person-circle-outline", iconActive: "person-circle" }
};

function Tabs() {
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  const badges = { Matches: unreadMessages, Notifications: notifications.data?.unreadCount ?? 0 };

  return (
    <Tab.Navigator
      screenOptions={{ headerShown: false }}
      tabBar={(props) => <BottomNavigation {...props} badges={badges} meta={tabMeta} />}
    >
      <Tab.Screen component={HomeScreen} name="Discover" />
      <Tab.Screen component={MatchesScreen} name="Matches" />
      <Tab.Screen component={NotificationsScreen} name="Notifications" />
      <Tab.Screen component={ProfileScreen} name="Me" />
    </Tab.Navigator>
  );
}

/** Announces a match made by the other person (ours are celebrated by the discover screen itself). */
function MatchToaster() {
  const { dismissMatch, incomingMatch, isMatchCelebrated } = useRealtime();

  useEffect(() => {
    if (!incomingMatch) {
      return;
    }
    const timer = setTimeout(() => {
      if (!isMatchCelebrated(incomingMatch.id)) {
        showToast(
          {
            message: `C'est un match avec ${incomingMatch.firstName} !`,
            tone: "success",
            actionLabel: "Écrire",
            onAction: () => {
              if (navigationRef.isReady()) {
                navigationRef.navigate("Thread", {
                  conversationId: incomingMatch.conversationId,
                  peerId: incomingMatch.userId,
                  peerName: incomingMatch.firstName
                });
              }
            }
          },
          8000
        );
      }
      dismissMatch();
    }, 700);
    return () => clearTimeout(timer);
  }, [dismissMatch, incomingMatch, isMatchCelebrated]);

  return null;
}

function SignedInStack() {
  const profile = useMyProfile();

  if (profile.isError && !profile.data) {
    return (
      <SafeAreaLayout>
        <ErrorView error={profile.error} onRetry={() => void profile.refetch()} />
      </SafeAreaLayout>
    );
  }
  if (!profile.data) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement de votre profil…" />
      </SafeAreaLayout>
    );
  }

  // Until the profile, preferences and a photo exist, nothing else is reachable.
  if (!profile.data.onboarding.complete) {
    return (
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        <Stack.Screen component={OnboardingScreen} name="Onboarding" />
      </Stack.Navigator>
    );
  }

  return (
    <>
      <MatchToaster />
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        <Stack.Screen component={Tabs} name="Tabs" />
        <Stack.Screen component={ChatScreen} name="Thread" />
        <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" />
        <Stack.Screen component={EditProfileScreen} name="EditProfile" />
        <Stack.Screen component={PhotosScreen} name="Photos" />
        <Stack.Screen component={PreferencesScreen} name="Preferences" />
        <Stack.Screen component={SettingsScreen} name="Settings" />
        <Stack.Screen component={BlockedScreen} name="Blocked" />
        <Stack.Screen component={SecurityScreen} name="Security" />
        <Stack.Screen component={ReportScreen} name="Report" options={{ presentation: "modal" }} />
      </Stack.Navigator>
    </>
  );
}

export function RootNavigator() {
  const palette = useTheme();
  const { retry, status } = useAuth();

  const base = palette.isDark ? DarkTheme : DefaultTheme;
  const theme: Theme = {
    ...base,
    dark: palette.isDark,
    colors: {
      ...base.colors,
      primary: palette.primary,
      background: palette.background,
      card: palette.surface,
      text: palette.text,
      border: palette.border,
      notification: palette.primary
    }
  };

  if (status === "booting") {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Ouverture de votre session…" />
      </SafeAreaLayout>
    );
  }
  if (status === "offline") {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de joindre le serveur. Vérifiez votre connexion." onRetry={retry} title="Hors ligne" />
      </SafeAreaLayout>
    );
  }

  return (
    <NavigationContainer ref={navigationRef} theme={theme}>
      {status === "signedIn" ? (
        <RealtimeProvider>
          <SignedInStack />
        </RealtimeProvider>
      ) : (
        <Stack.Navigator screenOptions={{ headerShown: false }}>
          <Stack.Screen component={AuthScreen} name="Auth" />
          <Stack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        </Stack.Navigator>
      )}
    </NavigationContainer>
  );
}
