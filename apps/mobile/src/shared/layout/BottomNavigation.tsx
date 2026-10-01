import React from "react";
import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import ChatScreen from "../../screens/ChatScreen";
import HomeScreen from "../../screens/HomeScreen";
import NotificationsScreen from "../../screens/NotificationsScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import { useConversations, useNotifications } from "../../hooks/useData";
import type { TabParamList } from "../../navigation/types";
import { useTheme } from "../ui";

const Tab = createBottomTabNavigator<TabParamList>();

const icons: Record<keyof TabParamList, [React.ComponentProps<typeof Ionicons>["name"], React.ComponentProps<typeof Ionicons>["name"]]> = {
  Discover: ["flame", "flame-outline"],
  Messages: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Profile: ["person", "person-outline"]
};

/** The four main tabs; unread counts come from the shared query cache. */
export function MainTabs() {
  const { colors } = useTheme();
  const unreadMessages = useConversations().data?.totalUnread ?? 0;
  const unreadNotifications = useNotifications().data?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      initialRouteName="Discover"
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.primaryForeground },
        tabBarIcon: ({ focused, color, size }) => <Ionicons color={color} name={icons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tab.Screen component={HomeScreen} name="Discover" options={{ title: "Découvrir" }} />
      <Tab.Screen
        component={ChatScreen}
        name="Messages"
        options={{ title: "Messages", tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ title: "Notifications", tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined }}
      />
      <Tab.Screen component={ProfileScreen} name="Profile" options={{ title: "Profil" }} />
    </Tab.Navigator>
  );
}
