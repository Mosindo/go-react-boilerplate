import React from "react";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import BlockedUsersScreen from "../screens/BlockedUsersScreen";
import ConversationScreen from "../screens/ConversationScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { MainTabs } from "../shared/layout/BottomNavigation";
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
      card: colors.background,
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
          headerTintColor: colors.text,
          headerStyle: { backgroundColor: colors.background },
          headerShadowVisible: false,
          headerBackTitleVisible: false,
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <Stack.Screen component={MainTabs} name="Tabs" options={{ headerShown: false }} />
        <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "Profil" }} />
        <Stack.Screen component={ConversationScreen} name="Conversation" options={{ title: "Conversation" }} />
        <Stack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
        <Stack.Screen component={SettingsScreen} name="Settings" options={{ title: "Réglages" }} />
        <Stack.Screen component={BlockedUsersScreen} name="BlockedUsers" options={{ title: "Personnes bloquées" }} />
      </Stack.Navigator>
    </NavigationContainer>
  );
}
