import React from "react";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useConversationsQuery, useNotificationsQuery } from "../hooks/useFeeds";
import BlockedUsersScreen from "../screens/BlockedUsersScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import HomeScreen from "../screens/HomeScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ProfileScreen from "../screens/ProfileScreen";
import { BottomNavigation } from "../shared/layout";
import type { RootStackParamList, TabParamList } from "./types";

const Tabs = createBottomTabNavigator<TabParamList>();
const Stack = createNativeStackNavigator<RootStackParamList>();

function TabNavigator() {
  const conversations = useConversationsQuery();
  const notifications = useNotificationsQuery();
  const unreadMessages = (conversations.data?.pages ?? []).flatMap((page) => page.conversations).reduce((sum, c) => sum + c.unreadCount, 0);
  const unreadActivity = notifications.data?.pages[0]?.unreadCount ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={{ headerShown: false }}
      tabBar={(props) => <BottomNavigation {...props} badges={{ Messages: unreadMessages, Notifications: unreadActivity }} />}
    >
      <Tabs.Screen component={HomeScreen} name="Discover" />
      <Tabs.Screen component={MessagesScreen} name="Messages" />
      <Tabs.Screen component={NotificationsScreen} name="Notifications" />
      <Tabs.Screen component={ProfileScreen} name="Me" />
    </Tabs.Navigator>
  );
}

export function MainNavigator() {
  return (
    <Stack.Navigator screenOptions={{ headerShown: false }}>
      <Stack.Screen component={TabNavigator} name="Tabs" />
      <Stack.Screen component={ChatScreen} name="Chat" />
      <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" />
      <Stack.Screen component={EditProfileScreen} name="EditProfile" />
      <Stack.Screen component={PreferencesScreen} name="Preferences" />
      <Stack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
    </Stack.Navigator>
  );
}
