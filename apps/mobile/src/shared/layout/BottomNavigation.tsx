import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import HomeScreen from "../../screens/HomeScreen";
import LikesScreen from "../../screens/LikesScreen";
import MatchesScreen from "../../screens/MatchesScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import { useMatches } from "../../hooks/queries";
import type { TabParamList } from "../../navigation/types";
import { useTheme } from "../ui";

const Tab = createBottomTabNavigator<TabParamList>();

type IconName = keyof typeof Ionicons.glyphMap;

const ICONS: Record<keyof TabParamList, { active: IconName; inactive: IconName; label: string }> = {
  Discover: { active: "compass", inactive: "compass-outline", label: "Découvrir" },
  Likes: { active: "heart", inactive: "heart-outline", label: "Ils vous aiment" },
  Messages: { active: "chatbubbles", inactive: "chatbubbles-outline", label: "Messages" },
  Me: { active: "person-circle", inactive: "person-circle-outline", label: "Profil" }
};

export function BottomNavigation() {
  const { colors } = useTheme();
  const matches = useMatches();
  const unread = matches.data?.unreadTotal ?? 0;

  return (
    <Tab.Navigator
      initialRouteName="Discover"
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarLabel: ICONS[route.name].label,
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons color={color} name={focused ? ICONS[route.name].active : ICONS[route.name].inactive} size={size} />
        )
      })}
    >
      <Tab.Screen component={HomeScreen} name="Discover" options={{ tabBarLabel: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen component={LikesScreen} name="Likes" options={{ tabBarLabel: "Likes", tabBarTestID: "tab-likes" }} />
      <Tab.Screen
        component={MatchesScreen}
        name="Messages"
        options={{ tabBarBadge: unread > 0 ? unread : undefined, tabBarTestID: "tab-messages" }}
      />
      <Tab.Screen component={ProfileScreen} name="Me" options={{ tabBarLabel: "Profil", tabBarTestID: "tab-profile" }} />
    </Tab.Navigator>
  );
}
