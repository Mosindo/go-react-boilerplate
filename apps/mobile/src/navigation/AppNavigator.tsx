import React, { type ReactNode } from "react";
import { Text as RNText } from "react-native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import { AccessTokenProvider } from "../hooks/useAccessToken";
import { useAuth, useMe } from "../hooks/useAuth";
import { useUnreadMessageCount } from "../hooks/useMatches";
import { useUnreadNotificationCount } from "../hooks/useNotifications";
import { RealtimeBridge } from "../hooks/useRealtime";
import AboutScreen from "../screens/AboutScreen";
import BlockedUsersScreen from "../screens/BlockedUsersScreen";
import ChangePasswordScreen from "../screens/ChangePasswordScreen";
import ChatScreen from "../screens/ChatScreen";
import DeleteAccountScreen from "../screens/DeleteAccountScreen";
import DiscoverScreen from "../screens/DiscoverScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import MatchCelebrationScreen from "../screens/MatchCelebrationScreen";
import MatchesScreen from "../screens/MatchesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileScreen from "../screens/ProfileScreen";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { Button } from "../shared/ui";
import { SafeAreaLayout } from "../shared/layout";
import { useTheme } from "../theme";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function Themed({ children }: { children: ReactNode }) {
  const palette = useTheme();
  const base = palette.isDark ? DarkTheme : DefaultTheme;
  const theme: Theme = {
    ...base,
    colors: {
      ...base.colors,
      primary: palette.primary,
      background: palette.background,
      card: palette.surface,
      text: palette.text,
      border: palette.border,
      notification: palette.primary
    }
  };
  return <NavigationContainer theme={theme}>{children}</NavigationContainer>;
}

function tabGlyph(glyph: string) {
  return function TabIcon({ color }: { color: string }) {
    return (
      <RNText accessibilityElementsHidden importantForAccessibility="no" style={{ color, fontSize: 20 }}>
        {glyph}
      </RNText>
    );
  };
}

function MainTabs() {
  const theme = useTheme();
  const unreadMessages = useUnreadMessageCount();
  const unreadNotifications = useUnreadNotificationCount();
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.textMuted,
        tabBarStyle: { backgroundColor: theme.surface, borderTopColor: theme.border },
        tabBarBadgeStyle: { backgroundColor: theme.primary, color: theme.onPrimary }
      }}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ tabBarIcon: tabGlyph("✦") }} />
      <Tabs.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          tabBarIcon: tabGlyph("♡"),
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarAccessibilityLabel: unreadMessages > 0 ? `Matches, ${unreadMessages} unread` : "Matches"
        }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          tabBarIcon: tabGlyph("✉"),
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarAccessibilityLabel:
            unreadNotifications > 0 ? `Notifications, ${unreadNotifications} unread` : "Notifications"
        }}
      />
      <Tabs.Screen component={ProfileScreen} name="Profile" options={{ tabBarIcon: tabGlyph("☺") }} />
    </Tabs.Navigator>
  );
}

function MainNavigator({ userId }: { userId: string }) {
  return (
    <Themed>
      <RealtimeBridge myUserId={userId} />
      <MainStack.Navigator screenOptions={{ headerBackTitleVisible: false }}>
        <MainStack.Screen component={MainTabs} name="Tabs" options={{ headerShown: false }} />
        <MainStack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
        <MainStack.Screen
          component={MatchCelebrationScreen}
          name="MatchCelebration"
          options={{ headerShown: false, presentation: "fullScreenModal", gestureEnabled: false }}
        />
        <MainStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Edit profile" }} />
        <MainStack.Screen component={PhotosScreen} name="Photos" options={{ title: "Photos" }} />
        <MainStack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Who I want to meet" }} />
        <MainStack.Screen component={BlockedUsersScreen} name="BlockedUsers" options={{ title: "Blocked people" }} />
        <MainStack.Screen
          component={ChangePasswordScreen}
          name="ChangePassword"
          options={{ title: "Change password" }}
        />
        <MainStack.Screen component={DeleteAccountScreen} name="DeleteAccount" options={{ title: "Delete account" }} />
        <MainStack.Screen component={AboutScreen} name="About" options={{ title: "About and privacy" }} />
      </MainStack.Navigator>
    </Themed>
  );
}

function SignedIn() {
  const meQuery = useMe();
  const { signOut } = useAuth();
  if (meQuery.isPending) {
    return <LoadingView />;
  }
  if (!meQuery.data) {
    return (
      <SafeAreaLayout>
        <ErrorView message={errorMessage(meQuery.error)} onRetry={() => void meQuery.refetch()} />
        <Button label="Sign out" onPress={() => void signOut()} variant="ghost" />
      </SafeAreaLayout>
    );
  }
  return (
    <AccessTokenProvider>
      {meQuery.data.profileComplete ? <MainNavigator userId={meQuery.data.id} /> : <OnboardingScreen />}
    </AccessTokenProvider>
  );
}

function SignedOut() {
  return (
    <Themed>
      <AuthStack.Navigator screenOptions={{ headerBackTitleVisible: false, headerShadowVisible: false, title: "" }}>
        <AuthStack.Screen component={WelcomeScreen} name="Welcome" options={{ headerShown: false }} />
        <AuthStack.Screen component={RegisterScreen} name="Register" />
        <AuthStack.Screen component={LoginScreen} name="Login" />
        <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
      </AuthStack.Navigator>
    </Themed>
  );
}

export function AppNavigator() {
  const { status, bootError, retryBoot } = useAuth();
  if (status === "booting") {
    return (
      <SafeAreaLayout>
        <LoadingView label="Restoring your session" />
      </SafeAreaLayout>
    );
  }
  if (status === "bootError") {
    return (
      <SafeAreaLayout>
        <ErrorView message={bootError ?? "We could not reach the server."} onRetry={retryBoot} title="Can't connect" />
      </SafeAreaLayout>
    );
  }
  return status === "signedIn" ? <SignedIn /> : <SignedOut />;
}
