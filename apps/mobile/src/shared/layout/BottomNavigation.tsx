import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { useQuery } from "@tanstack/react-query";
import { listConversations, listNotifications, queryKeys } from "../../api/platform";
import type { TabParamList } from "../../navigation/types";
import HomeScreen from "../../screens/HomeScreen";
import MatchesScreen from "../../screens/MatchesScreen";
import NotificationsScreen from "../../screens/NotificationsScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import { colors } from "../ui";

const Tab = createBottomTabNavigator<TabParamList>();

/** Bottom tabs of the signed-in app. Badges reuse the cached conversation and notification queries. */
export function BottomNavigation() {
  const conversations = useQuery({ queryKey: queryKeys.conversations, queryFn: listConversations });
  const notifications = useQuery({ queryKey: queryKeys.notifications, queryFn: listNotifications });
  const unreadMessages = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  const unreadActivity = notifications.data?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      initialRouteName="Discover"
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.backgroundElevated, borderTopColor: colors.border },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.primaryForeground }
      }}
    >
      <Tab.Screen component={HomeScreen} name="Discover" options={{ tabBarTestID: "tab-discover", tabBarIcon: () => null, tabBarIconStyle: { display: "none" } }} />
      <Tab.Screen
        component={MatchesScreen}
        name="Matches"
        options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined, tabBarTestID: "tab-matches", tabBarIcon: () => null, tabBarIconStyle: { display: "none" } }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Activity"
        options={{ tabBarBadge: unreadActivity > 0 ? unreadActivity : undefined, tabBarTestID: "tab-activity", tabBarIcon: () => null, tabBarIconStyle: { display: "none" } }}
      />
      <Tab.Screen component={ProfileScreen} name="Profile" options={{ tabBarTestID: "tab-profile", tabBarIcon: () => null, tabBarIconStyle: { display: "none" } }} />
    </Tab.Navigator>
  );
}
