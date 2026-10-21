import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import React from "react";
import { useUnreadCounts } from "../hooks/queries";
import { HomeScreen } from "../screens/discover/HomeScreen";
import { MessagesScreen } from "../screens/messages/MessagesScreen";
import { ProfileScreen } from "../screens/profile/ProfileScreen";
import { useTheme } from "../theme";
import type { TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();

export function MainTabs() {
  const theme = useTheme();
  const unread = useUnreadCounts();

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.textMuted,
        tabBarStyle: { backgroundColor: theme.background, borderTopColor: theme.border, height: 62, paddingTop: 6, paddingBottom: 8 },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarIcon: ({ color, size, focused }) => {
          const icons = {
            Discover: focused ? "flame" : "flame-outline",
            Messages: focused ? "chatbubbles" : "chatbubbles-outline",
            Profile: focused ? "person" : "person-outline"
          } as const;
          return <Ionicons name={icons[route.name]} size={size} color={color} />;
        }
      })}
    >
      <Tab.Screen name="Discover" component={HomeScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarTestID: "tab-messages",
          tabBarBadge: unread.messages > 0 ? unread.messages : undefined,
          tabBarBadgeStyle: { backgroundColor: theme.primary, color: theme.primaryText }
        }}
      />
      <Tab.Screen name="Profile" component={ProfileScreen} options={{ title: "Profil", tabBarTestID: "tab-profile" }} />
    </Tab.Navigator>
  );
}
