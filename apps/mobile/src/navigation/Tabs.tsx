import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DiscoverScreen } from "../features/discovery/DiscoverScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { useChats } from "../features/matches/useChats";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { useNotifications } from "../features/notifications/useNotifications";
import { MeScreen } from "../features/profile/MeScreen";
import { useTheme } from "../theme";
import { Icon, type IconName } from "../ui";
import type { TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();

const icons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

export function Tabs() {
  const theme = useTheme();
  const chats = useChats();
  const notifications = useNotifications();
  const unreadMessages = chats.data?.pages.flat().reduce((sum, c) => sum + c.unreadCount, 0) ?? 0;
  const newMatches = chats.data?.pages.flat().filter((c) => !c.lastMessage).length ?? 0;
  const unreadNotifications = notifications.data?.pages[0]?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.colors.primary,
        tabBarInactiveTintColor: theme.colors.textSubtle,
        tabBarStyle: {
          backgroundColor: theme.colors.surface,
          borderTopColor: theme.colors.border,
          height: 64,
          paddingTop: 6,
          paddingBottom: 8
        },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: theme.colors.primary, color: theme.colors.textOnPrimary, fontSize: 10 },
        tabBarIcon: ({ focused, color }) => <Icon name={icons[route.name][focused ? 0 : 1]} size={24} color={color} />
      })}
    >
      <Tab.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        name="Matches"
        component={MatchesScreen}
        options={{
          title: "Matchs",
          tabBarTestID: "tab-matches",
          tabBarBadge: unreadMessages > 0 ? (unreadMessages > 99 ? "99+" : unreadMessages) : newMatches > 0 ? "•" : undefined
        }}
      />
      <Tab.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{
          title: "Activité",
          tabBarTestID: "tab-notifications",
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined
        }}
      />
      <Tab.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tab.Navigator>
  );
}
