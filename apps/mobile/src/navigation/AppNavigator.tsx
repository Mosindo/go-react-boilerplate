import React from "react";
import { Ionicons } from "@expo/vector-icons";
import { DefaultTheme, DarkTheme, NavigationContainer } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useConversations, useNotifications } from "../hooks/queries";
import ConversationScreen from "../screens/ConversationScreen";
import DiscoverScreen from "../screens/DiscoverScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ProfileScreen from "../screens/ProfileScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { useTheme } from "../theme";
import type { RootStackParamList, TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();
const Stack = createNativeStackNavigator<RootStackParamList>();

const TAB_ICONS: Record<keyof TabParamList, { active: keyof typeof Ionicons.glyphMap; inactive: keyof typeof Ionicons.glyphMap }> = {
  Discover: { active: "flame", inactive: "flame-outline" },
  Messages: { active: "chatbubbles", inactive: "chatbubbles-outline" },
  Notifications: { active: "notifications", inactive: "notifications-outline" },
  Profile: { active: "person", inactive: "person-outline" }
};

function Tabs() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  const unreadNotifications = notifications.data?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      initialRouteName="Discover"
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.primaryForeground },
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons color={color} name={focused ? TAB_ICONS[route.name].active : TAB_ICONS[route.name].inactive} size={size} />
        )
      })}
    >
      <Tab.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        component={MessagesScreen}
        name="Messages"
        options={{ title: "Messages", tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined, tabBarTestID: "tab-messages" }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          title: "Alertes",
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarTestID: "tab-notifications"
        }}
      />
      <Tab.Screen component={ProfileScreen} name="Profile" options={{ title: "Profil", tabBarTestID: "tab-profile" }} />
    </Tab.Navigator>
  );
}

export function AppNavigator() {
  const { colors, scheme } = useTheme();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.primary,
      text: colors.text,
      notification: colors.primary
    }
  };
  const header = {
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitleVisible: false
  } as const;

  return (
    <NavigationContainer theme={theme}>
      <Stack.Navigator screenOptions={{ ...header, contentStyle: { backgroundColor: colors.background } }}>
        <Stack.Screen component={Tabs} name="Tabs" options={{ headerShown: false }} />
        <Stack.Screen component={ConversationScreen} name="Conversation" options={{ headerShown: false }} />
        <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ headerShown: false }} />
        <Stack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
        <Stack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Préférences" }} />
        <Stack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
        <Stack.Screen component={SettingsScreen} name="Settings" options={{ title: "Confidentialité et compte" }} />
      </Stack.Navigator>
    </NavigationContainer>
  );
}
