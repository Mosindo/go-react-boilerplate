import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import ChatScreen from "../../screens/ChatScreen";
import HomeScreen from "../../screens/HomeScreen";
import MatchesScreen from "../../screens/MatchesScreen";
import NotificationsScreen from "../../screens/NotificationsScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import { useNotifications, useUnreadMessages } from "../../hooks/queries";
import { useRealtime } from "../../hooks/useRealtime";
import type { TabParamList } from "../../navigation/types";
import { colors, shadows, spacing } from "../ui";

const Tab = createBottomTabNavigator<TabParamList>();

type IconName = React.ComponentProps<typeof Ionicons>["name"];

const ICONS: Record<keyof TabParamList, { active: IconName; idle: IconName; label: string }> = {
  Home: { active: "flame", idle: "flame-outline", label: "Découvrir" },
  Matches: { active: "heart", idle: "heart-outline", label: "Matchs" },
  Chat: { active: "chatbubbles", idle: "chatbubbles-outline", label: "Messages" },
  Notifications: { active: "notifications", idle: "notifications-outline", label: "Alertes" },
  Profile: { active: "person", idle: "person-outline", label: "Profil" }
};

export function BottomNavigation() {
  const { connected } = useRealtime();
  const unreadMessages = useUnreadMessages(connected).data ?? 0;
  const unreadAlerts = useNotifications(connected).data?.unread ?? 0;

  const badge = (count: number) => (count > 0 ? (count > 99 ? "99+" : count) : undefined);

  return (
    <Tab.Navigator
      initialRouteName="Home"
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarHideOnKeyboard: true,
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons color={color} name={focused ? ICONS[route.name].active : ICONS[route.name].idle} size={size} />
        ),
        tabBarLabel: ICONS[route.name].label,
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.inverse },
        tabBarStyle: {
          position: "absolute",
          left: spacing.lg,
          right: spacing.lg,
          bottom: spacing.md,
          height: 68,
          paddingTop: spacing.xs,
          backgroundColor: colors.backgroundElevated,
          borderTopColor: colors.border,
          borderTopWidth: 1,
          borderRadius: 22,
          ...shadows.floating
        },
        tabBarLabelStyle: { paddingBottom: spacing.xs, fontSize: 11, fontWeight: "600" },
        tabBarItemStyle: { paddingVertical: spacing.xs }
      })}
      sceneContainerStyle={{ backgroundColor: colors.background }}
    >
      <Tab.Screen component={HomeScreen} name="Home" options={{ tabBarTestID: "tab-home" }} />
      <Tab.Screen component={MatchesScreen} name="Matches" options={{ tabBarTestID: "tab-matches" }} />
      <Tab.Screen component={ChatScreen} name="Chat" options={{ tabBarBadge: badge(unreadMessages), tabBarTestID: "tab-chat" }} />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ tabBarBadge: badge(unreadAlerts), tabBarTestID: "tab-notifications" }}
      />
      <Tab.Screen component={ProfileScreen} name="Profile" options={{ tabBarTestID: "tab-profile" }} />
    </Tab.Navigator>
  );
}
