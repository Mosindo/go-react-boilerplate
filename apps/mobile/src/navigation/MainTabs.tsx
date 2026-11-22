import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { useChats, useLikesReceived } from "../hooks/queries";
import ChatsScreen from "../screens/chat/ChatsScreen";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import LikesScreen from "../screens/likes/LikesScreen";
import MyProfileScreen from "../screens/profile/MyProfileScreen";
import { colors } from "../ui";
import type { TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();

const icons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
  Discover: ["flame", "flame-outline"],
  Likes: ["heart", "heart-outline"],
  Messages: ["chatbubbles", "chatbubbles-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

export function MainTabs() {
  const chats = useChats();
  const likes = useLikesReceived();
  const unread = chats.data?.totalUnread ?? 0;
  const likeCount = likes.data?.length ?? 0;

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.border,
          height: 64,
          paddingBottom: 8,
          paddingTop: 6
        },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.textOnAccent, fontSize: 10 },
        tabBarIcon: ({ focused, color, size }) => <Ionicons color={color} name={icons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tab.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        component={LikesScreen}
        name="Likes"
        options={{ title: "J'aime", tabBarBadge: likeCount > 0 ? likeCount : undefined, tabBarTestID: "tab-likes" }}
      />
      <Tab.Screen
        component={ChatsScreen}
        name="Messages"
        options={{ title: "Messages", tabBarBadge: unread > 0 ? (unread > 99 ? "99+" : unread) : undefined, tabBarTestID: "tab-messages" }}
      />
      <Tab.Screen component={MyProfileScreen} name="Me" options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tab.Navigator>
  );
}
