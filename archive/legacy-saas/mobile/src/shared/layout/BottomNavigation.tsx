import React from "react";
import { DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { type AuthUser } from "../../api/auth";
import ChatScreen from "../../screens/ChatScreen";
import HomeScreen from "../../screens/HomeScreen";
import NotificationsScreen from "../../screens/NotificationsScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import { colors, shadows, spacing } from "../ui";

export type RootTabParamList = {
  Home: undefined;
  Chat: undefined;
  Notifications: undefined;
  Profile: undefined;
};

type BottomNavigationProps = {
  accessToken: string;
  user: AuthUser;
};

const Tab = createBottomTabNavigator<RootTabParamList>();
const navigationTheme = {
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

export function BottomNavigation({ accessToken, user }: BottomNavigationProps) {
  return (
    <NavigationContainer theme={navigationTheme}>
      <Tab.Navigator
        initialRouteName="Home"
        sceneContainerStyle={{
          backgroundColor: colors.background
        }}
        screenOptions={{
          headerShown: false,
          tabBarActiveTintColor: colors.primary,
          tabBarInactiveTintColor: colors.textMuted,
          tabBarStyle: {
            position: "absolute",
            left: spacing.lg,
            right: spacing.lg,
            bottom: spacing.md,
            height: 68,
            paddingTop: spacing.xs,
            backgroundColor: colors.backgroundElevated,
            borderTopColor: colors.border,
            borderTopWidth: 1,
            borderRadius: 22,
            ...shadows.floating
          },
          tabBarLabelStyle: {
            paddingBottom: spacing.xs,
            fontSize: 12,
            fontWeight: "600"
          },
          tabBarItemStyle: {
            paddingVertical: spacing.xs
          }
        }}
      >
        <Tab.Screen name="Home">
          {() => <HomeScreen currentUserId={user.id} token={accessToken} />}
        </Tab.Screen>
        <Tab.Screen name="Chat">
          {() => <ChatScreen currentUserId={user.id} token={accessToken} />}
        </Tab.Screen>
        <Tab.Screen name="Notifications">
          {() => <NotificationsScreen token={accessToken} />}
        </Tab.Screen>
        <Tab.Screen name="Profile">
          {() => <ProfileScreen token={accessToken} user={user} />}
        </Tab.Screen>
      </Tab.Navigator>
    </NavigationContainer>
  );
}
