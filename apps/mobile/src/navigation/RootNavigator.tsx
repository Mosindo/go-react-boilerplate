import React from "react";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import BlockedUsersScreen from "../screens/BlockedUsersScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ReportScreen from "../screens/ReportScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { BottomNavigation } from "../shared/layout/BottomNavigation";
import { useTheme } from "../shared/ui";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

export function RootNavigator() {
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

  return (
    <NavigationContainer theme={theme}>
      <Stack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: colors.surface },
          headerTintColor: colors.text,
          headerShadowVisible: false,
          headerBackTitleVisible: false,
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <Stack.Screen component={BottomNavigation} name="Tabs" options={{ headerShown: false }} />
        <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "" }} />
        <Stack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
        <Stack.Screen component={NotificationsScreen} name="Notifications" options={{ title: "Notifications" }} />
        <Stack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
        <Stack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
        <Stack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Préférences de rencontre" }} />
        <Stack.Screen component={SettingsScreen} name="Settings" options={{ title: "Paramètres" }} />
        <Stack.Screen component={BlockedUsersScreen} name="BlockedUsers" options={{ title: "Personnes bloquées" }} />
        <Stack.Screen component={ReportScreen} name="Report" options={{ title: "Signaler", presentation: "modal" }} />
      </Stack.Navigator>
    </NavigationContainer>
  );
}
