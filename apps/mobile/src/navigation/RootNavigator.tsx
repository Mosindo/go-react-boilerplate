import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { NavigationContainer, createNavigationContainerRef, DefaultTheme, DarkTheme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React from "react";
import { errorMessage } from "../api/client";
import { MatchProvider } from "../components/MatchOverlay";
import { useMyProfile, useNotifications, useUnreadMessages } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import AuthScreen from "../screens/AuthScreen";
import BlockedScreen from "../screens/BlockedScreen";
import ChatScreen from "../screens/ChatScreen";
import DiscoverScreen from "../screens/DiscoverScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import ProfileScreen from "../screens/ProfileScreen";
import ReportScreen from "../screens/ReportScreen";
import SettingsScreen from "../screens/SettingsScreen";
import UserProfileScreen from "../screens/UserProfileScreen";
import { useTheme } from "../theme";
import { ErrorState, LoadingState } from "../ui";
import type { MainStackParamList, TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();
const Stack = createNativeStackNavigator<MainStackParamList>();
const navigationRef = createNavigationContainerRef<MainStackParamList>();

const ICONS: Record<keyof TabParamList, { on: keyof typeof Ionicons.glyphMap; off: keyof typeof Ionicons.glyphMap; label: string }> = {
  Discover: { on: "flame", off: "flame-outline", label: "Découvrir" },
  Messages: { on: "chatbubbles", off: "chatbubbles-outline", label: "Messages" },
  Notifications: { on: "notifications", off: "notifications-outline", label: "Activité" },
  Profile: { on: "person", off: "person-outline", label: "Profil" }
};

function Tabs() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessages();
  const notifications = useNotifications();
  const unreadNotifications = notifications.data?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabel: ICONS[route.name].label,
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons name={focused ? ICONS[route.name].on : ICONS[route.name].off} size={size} color={color} />
        )
      })}
    >
      <Tab.Screen name="Discover" component={DiscoverScreen} />
      <Tab.Screen name="Messages" component={MessagesScreen} options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined }} />
      <Tab.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{ tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined }}
      />
      <Tab.Screen name="Profile" component={ProfileScreen} />
    </Tab.Navigator>
  );
}

function MainStack() {
  const { colors } = useTheme();
  const header = {
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour"
  };
  return (
    <Stack.Navigator screenOptions={header}>
      <Stack.Screen name="Tabs" component={Tabs} options={{ headerShown: false }} />
      <Stack.Screen name="Chat" component={ChatScreen} />
      <Stack.Screen name="UserProfile" component={UserProfileScreen} options={{ title: "Profil" }} />
      <Stack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier le profil" }} />
      <Stack.Screen name="Settings" component={SettingsScreen} options={{ title: "Réglages" }} />
      <Stack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Personnes bloquées" }} />
      <Stack.Screen name="Report" component={ReportScreen} options={{ title: "Signalement" }} />
    </Stack.Navigator>
  );
}

/** Signed-in area: onboarding until the profile is complete, then the app. */
function SignedIn() {
  const profile = useMyProfile();
  if (profile.isLoading) {
    return <LoadingState />;
  }
  if (profile.isError || !profile.data) {
    return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  }
  if (!profile.data.onboardingComplete) {
    return <OnboardingScreen />;
  }
  return (
    <MatchProvider
      selfPhoto={profile.data.photos[0]?.thumbUrl}
      onOpenChat={(match) =>
        navigationRef.isReady() &&
        navigationRef.navigate("Chat", { matchId: match.id, userId: match.profile.userId, name: match.profile.firstName })
      }
    >
      <MainStack />
    </MatchProvider>
  );
}

export function RootNavigator() {
  const { user, isBooting } = useAuth();
  const { colors, isDark } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      text: colors.text,
      border: colors.border,
      primary: colors.primary
    }
  };

  if (isBooting) {
    return <LoadingState label="Chargement…" />;
  }
  return (
    <NavigationContainer ref={navigationRef} theme={theme}>
      {user ? <SignedIn key={user.id} /> : <AuthScreen />}
    </NavigationContainer>
  );
}
