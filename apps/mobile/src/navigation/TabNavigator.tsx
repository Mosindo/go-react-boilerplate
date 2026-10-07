import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import React from "react";
import { strings } from "../i18n/strings";
import { useUnreadMessagesCount, useUnreadNotificationsCount } from "../realtime/RealtimeProvider";
import DiscoverScreen from "../screens/DiscoverScreen";
import MatchesScreen from "../screens/MatchesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import ProfileScreen from "../screens/ProfileScreen";
import { useTheme } from "../theme/useTheme";
import type { TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();

type IconName = React.ComponentProps<typeof Ionicons>["name"];

const ICONS: Record<keyof TabParamList, { active: IconName; inactive: IconName }> = {
  Discover: { active: "flame", inactive: "flame-outline" },
  Matches: { active: "chatbubbles", inactive: "chatbubbles-outline" },
  Notifications: { active: "notifications", inactive: "notifications-outline" },
  Profile: { active: "person", inactive: "person-outline" }
};

function badgeValue(count: number): string | number | undefined {
  if (count <= 0) {
    return undefined;
  }
  return count > 99 ? "99+" : count;
}

export default function TabNavigator() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessagesCount();
  const unreadNotifications = useUnreadNotificationsCount();

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.backgroundElevated, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.primaryForeground },
        tabBarIcon: ({ color, focused, size }) => (
          <Ionicons
            color={color}
            name={focused ? ICONS[route.name].active : ICONS[route.name].inactive}
            size={size}
          />
        )
      })}
    >
      <Tab.Screen component={DiscoverScreen} name="Discover" options={{ title: strings.tabs.discover }} />
      <Tab.Screen
        component={MatchesScreen}
        name="Matches"
        options={{ title: strings.tabs.matches, tabBarBadge: badgeValue(unreadMessages) }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ title: strings.tabs.notifications, tabBarBadge: badgeValue(unreadNotifications) }}
      />
      <Tab.Screen component={ProfileScreen} name="Profile" options={{ title: strings.tabs.profile }} />
    </Tab.Navigator>
  );
}
