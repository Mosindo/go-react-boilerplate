import React from "react";
import { StyleSheet, View } from "react-native";
import { NavigationContainer, DefaultTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { getNotifications } from "../api/dating";
import { useAuth } from "../auth/AuthContext";
import { useOwnProfile } from "../hooks/useOwnProfile";
import { queryKeys } from "../lib/queryKeys";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import ChatScreen from "../screens/ChatScreen";
import DiscoverScreen from "../screens/DiscoverScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import MeScreen from "../screens/MeScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import PrivacyScreen from "../screens/PrivacyScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { colors } from "../shared/ui";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const navTheme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    background: colors.background,
    card: colors.backgroundElevated,
    text: colors.text,
    primary: colors.primary,
    border: colors.border
  }
};

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function useUnreadNotifications(): number {
  const { data } = useQuery({ queryKey: queryKeys.notificationBadge, queryFn: () => getNotifications(), refetchInterval: 60_000 });
  return data?.unreadCount ?? 0;
}

function TabIcon({ label, focused }: { label: string; focused: boolean }) {
  return <View style={[styles.dot, focused ? styles.dotActive : null]} accessibilityLabel={label} />;
}

function TabNavigator() {
  const unread = useUnreadNotifications();
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarLabelStyle: { fontWeight: "600", fontSize: 12 },
        tabBarStyle: { backgroundColor: colors.backgroundElevated, borderTopColor: colors.border }
      }}
    >
      <Tabs.Screen
        component={DiscoverScreen}
        name="Discover"
        options={{ title: "Découvrir", tabBarIcon: ({ focused }) => <TabIcon focused={focused} label="Découvrir" /> }}
      />
      <Tabs.Screen
        component={MessagesScreen}
        name="Messages"
        options={{ title: "Messages", tabBarIcon: ({ focused }) => <TabIcon focused={focused} label="Messages" /> }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          title: "Alertes",
          tabBarBadge: unread > 0 ? unread : undefined,
          tabBarIcon: ({ focused }) => <TabIcon focused={focused} label="Alertes" />
        }}
      />
      <Tabs.Screen
        component={MeScreen}
        name="Me"
        options={{ title: "Profil", tabBarIcon: ({ focused }) => <TabIcon focused={focused} label="Profil" /> }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  return (
    <MainStack.Navigator
      screenOptions={{ headerTintColor: colors.text, headerBackTitle: "Retour", contentStyle: { backgroundColor: colors.background } }}
    >
      <MainStack.Screen component={TabNavigator} name="Tabs" options={{ headerShown: false }} />
      <MainStack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
      <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "Profil" }} />
      <MainStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
      <MainStack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
      <MainStack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Préférences" }} />
      <MainStack.Screen component={PrivacyScreen} name="Privacy" options={{ title: "Confidentialité et sécurité" }} />
    </MainStack.Navigator>
  );
}

/** Signed-in users without a complete profile are routed through onboarding. */
function SignedIn({ userId }: { userId: string }) {
  const profile = useOwnProfile();
  if (profile.isLoading) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement de votre profil…" />
      </SafeAreaLayout>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <SafeAreaLayout>
        <ErrorView actionLabel="Réessayer" message="Impossible de charger votre profil." onAction={() => void profile.refetch()} />
      </SafeAreaLayout>
    );
  }
  if (!profile.data.complete) {
    return <OnboardingScreen profile={profile.data} />;
  }
  return (
    <RealtimeProvider userId={userId}>
      <NavigationContainer theme={navTheme}>
        <MainNavigator />
      </NavigationContainer>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status, user } = useAuth();
  if (status === "booting") {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement…" />
      </SafeAreaLayout>
    );
  }
  if (status === "signedOut" || !user) {
    return (
      <NavigationContainer theme={navTheme}>
        <AuthStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
          <AuthStack.Screen component={LoginScreen} name="Login" />
          <AuthStack.Screen component={RegisterScreen} name="Register" />
          <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        </AuthStack.Navigator>
      </NavigationContainer>
    );
  }
  return <SignedIn key={user.id} userId={user.id} />;
}

const styles = StyleSheet.create({
  dot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.borderStrong },
  dotActive: { backgroundColor: colors.primary }
});
