import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import React, { useMemo } from "react";
import ChatsScreen, { useConversations } from "../screens/main/ConversationsScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import NotificationsScreen, { useNotifications } from "../screens/main/NotificationsScreen";
import ProfileScreen from "../screens/main/ProfileScreen";
import { useTheme } from "../theme/ThemeProvider";
import type { MainTabParamList } from "./types";

const Tab = createBottomTabNavigator<MainTabParamList>();

const ICONS = {
  Discover: ["flame", "flame-outline"],
  Chats: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person", "person-outline"]
} as const;

const LABELS: Record<keyof MainTabParamList, string> = {
  Discover: "Découvrir",
  Chats: "Messages",
  Notifications: "Activité",
  Me: "Profil"
};

export default function MainTabs() {
  const theme = useTheme();
  const conversations = useConversations();
  const notifications = useNotifications();

  const unreadMessages = useMemo(
    () =>
      conversations.data?.pages
        .flatMap((p) => p.conversations)
        .reduce((sum, c) => sum + c.unreadCount, 0) ?? 0,
    [conversations.data]
  );
  const unreadNotifications = notifications.data?.pages[0]?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.textMuted,
        tabBarStyle: { backgroundColor: theme.surface, borderTopColor: theme.border },
        tabBarBadgeStyle: { backgroundColor: theme.primary, color: "#FFFFFF" },
        tabBarLabel: LABELS[route.name],
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons color={color} name={ICONS[route.name][focused ? 0 : 1]} size={size} />
        )
      })}
    >
      <Tab.Screen component={DiscoverScreen} name="Discover" />
      <Tab.Screen
        component={ChatsScreen}
        name="Chats"
        options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined }}
      />
      <Tab.Screen component={ProfileScreen} name="Me" />
    </Tab.Navigator>
  );
}
