import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { useConversations, useMatches, useUnread } from "../hooks/data";
import type { TabParamList } from "../navigation/types";
import { TabIcon } from "../shared/layout";
import { useTheme } from "../theme";
import ConversationsScreen from "./ConversationsScreen";
import DiscoverScreen from "./DiscoverScreen";
import MatchesScreen from "./MatchesScreen";
import ProfileScreen from "./ProfileScreen";

const Tab = createBottomTabNavigator<TabParamList>();

/** Home of the signed-in app: the four primary tabs. */
export default function HomeScreen() {
  const { colors } = useTheme();
  const unread = useUnread();
  const matches = useMatches();
  useConversations();
  const newMatches = matches.data?.pages.flatMap((page) => page.matches).filter((match) => !match.hasMessages).length ?? 0;

  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, height: 62, paddingTop: 6, paddingBottom: 8 }
      }}
    >
      <Tab.Screen
        component={DiscoverScreen}
        name="Discover"
        options={{
          title: "Découvrir",
          tabBarIcon: (p) => <TabIcon {...p} focusedName="flame" name="flame-outline" />
        }}
      />
      <Tab.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          title: "Matchs",
          tabBarIcon: (p) => <TabIcon {...p} badge={newMatches} focusedName="heart" name="heart-outline" />
        }}
      />
      <Tab.Screen
        component={ConversationsScreen}
        name="Messages"
        options={{
          title: "Messages",
          tabBarIcon: (p) => <TabIcon {...p} badge={unread.data?.unreadConversations ?? 0} focusedName="chatbubbles" name="chatbubbles-outline" />
        }}
      />
      <Tab.Screen
        component={ProfileScreen}
        name="Me"
        options={{
          title: "Moi",
          tabBarIcon: (p) => <TabIcon {...p} focusedName="person" name="person-outline" />
        }}
      />
    </Tab.Navigator>
  );
}
