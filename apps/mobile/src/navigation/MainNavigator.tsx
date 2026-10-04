import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React from "react";
import { ChatScreen } from "../features/chat/ChatScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { ChangePasswordScreen } from "../features/profile/ChangePasswordScreen";
import { DeleteAccountScreen } from "../features/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { ManagePhotosScreen } from "../features/profile/ManagePhotosScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { ProfileDetailScreen } from "../features/profile/ProfileDetailScreen";
import { ProfileScreen } from "../features/profile/ProfileScreen";
import { BlockedUsersScreen } from "../features/safety/BlockedUsersScreen";
import { SafetyTipsScreen } from "../features/safety/SafetyTipsScreen";
import { useConversationUnreadCount } from "../hooks/useChat";
import { useNotificationUnreadCount } from "../hooks/useNotifications";
import { useRealtimeConnection } from "../hooks/useRealtime";
import { useColors } from "../theme";
import { Icon, type IconName } from "../ui";
import type { MainStackParamList, TabParamList } from "./types";

const Tabs = createBottomTabNavigator<TabParamList>();
const Stack = createNativeStackNavigator<MainStackParamList>();

const TAB_ICONS: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Profile: ["person", "person-outline"]
};

function TabNavigator() {
  const colors = useColors();
  const unreadChats = useConversationUnreadCount();
  const unreadNotifications = useNotificationUnreadCount();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, minHeight: 56 },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ focused, color, size }) => (
          <Icon color={color} name={TAB_ICONS[route.name][focused ? 0 : 1]} size={size} />
        )
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ tabBarAccessibilityLabel: "Discover" }} />
      <Tabs.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          tabBarBadge: unreadChats > 0 ? (unreadChats > 99 ? "99+" : unreadChats) : undefined,
          tabBarAccessibilityLabel: unreadChats > 0 ? `Matches, ${unreadChats} unread` : "Matches"
        }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          tabBarBadge: unreadNotifications > 0 ? (unreadNotifications > 99 ? "99+" : unreadNotifications) : undefined,
          tabBarAccessibilityLabel:
            unreadNotifications > 0 ? `Notifications, ${unreadNotifications} unread` : "Notifications"
        }}
      />
      <Tabs.Screen component={ProfileScreen} name="Profile" options={{ tabBarAccessibilityLabel: "Profile" }} />
    </Tabs.Navigator>
  );
}

export function MainNavigator() {
  const colors = useColors();
  useRealtimeConnection();
  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.text,
        headerTitleStyle: { fontWeight: "600" },
        headerShadowVisible: false,
        contentStyle: { backgroundColor: colors.background },
        headerBackTitleVisible: false
      }}
    >
      <Stack.Screen component={TabNavigator} name="Tabs" options={{ headerShown: false }} />
      <Stack.Screen component={ChatScreen} name="Chat" />
      <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "Profile" }} />
      <Stack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Edit profile" }} />
      <Stack.Screen component={ManagePhotosScreen} name="ManagePhotos" options={{ title: "Photos" }} />
      <Stack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Discovery preferences" }} />
      <Stack.Screen component={BlockedUsersScreen} name="BlockedUsers" options={{ title: "Blocked people" }} />
      <Stack.Screen component={ChangePasswordScreen} name="ChangePassword" options={{ title: "Change password" }} />
      <Stack.Screen component={DeleteAccountScreen} name="DeleteAccount" options={{ title: "Delete account" }} />
      <Stack.Screen component={SafetyTipsScreen} name="SafetyTips" options={{ title: "Safety tips" }} />
    </Stack.Navigator>
  );
}
