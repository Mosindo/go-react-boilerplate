import React from "react";
import { DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { MatchModal } from "../components/MatchModal";
import EditProfileScreen from "../screens/EditProfileScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import { BlockedScreen, PhotosScreen, PreferencesScreen, PrivacyScreen } from "../screens/SettingsScreens";
import ThreadScreen from "../screens/ThreadScreen";
import { BottomNavigation } from "../shared/layout";
import { colors } from "../shared/ui";
import { navigationRef, openThread } from "./ref";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

const theme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    background: colors.background,
    card: colors.backgroundElevated,
    border: colors.border,
    primary: colors.primary,
    text: colors.text,
    notification: colors.primary
  }
};

const settingsOptions = {
  headerShown: true,
  headerTitle: "",
  headerBackTitle: "Retour",
  headerShadowVisible: false,
  headerTintColor: colors.primary,
  headerStyle: { backgroundColor: colors.background }
} as const;

export function RootNavigator() {
  return (
    <>
      <NavigationContainer ref={navigationRef} theme={theme}>
        <Stack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
          <Stack.Screen component={BottomNavigation} name="Tabs" />
          <Stack.Screen component={ThreadScreen} name="Thread" />
          <Stack.Screen component={ProfileDetailScreen} name="ProfileDetail" />
          <Stack.Screen component={EditProfileScreen} name="EditProfile" options={settingsOptions} />
          <Stack.Screen component={PhotosScreen} name="Photos" options={settingsOptions} />
          <Stack.Screen component={PreferencesScreen} name="Preferences" options={settingsOptions} />
          <Stack.Screen component={PrivacyScreen} name="Privacy" options={settingsOptions} />
          <Stack.Screen component={BlockedScreen} name="Blocked" options={settingsOptions} />
        </Stack.Navigator>
      </NavigationContainer>
      <MatchModal onOpenChat={(matchId, userId) => void openThread(matchId, userId)} />
    </>
  );
}
