import React, { useEffect, useState } from "react";
import { Ionicons } from "@expo/vector-icons";
import { DefaultTheme, DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { errorMessage, useConversations, useNotifications, useProfile } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import AuthScreen from "../screens/AuthScreen";
import ChatScreen from "../screens/ChatScreen";
import DiscoverScreen from "../screens/DiscoverScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ForgotScreen from "../screens/ForgotScreen";
import MeScreen from "../screens/MeScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PhotosScreen from "../screens/PhotosScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import PrivacyScreen from "../screens/PrivacyScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { useSession } from "../state/session";
import { ErrorView, LoadingView, useTheme } from "../ui";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

type IconName = React.ComponentProps<typeof Ionicons>["name"];
const TAB_ICONS: Record<keyof TabParamList, { on: IconName; off: IconName; label: string }> = {
  Discover: { on: "flame", off: "flame-outline", label: "Découvrir" },
  Messages: { on: "chatbubbles", off: "chatbubbles-outline", label: "Messages" },
  Notifications: { on: "notifications", off: "notifications-outline", label: "Alertes" },
  Me: { on: "person", off: "person-outline", label: "Moi" }
};

function MainTabs() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  const unreadNotifications = notifications.data?.unreadCount ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => {
        const icon = TAB_ICONS[route.name];
        return {
          headerShown: false,
          tabBarActiveTintColor: colors.primary,
          tabBarInactiveTintColor: colors.textMuted,
          tabBarLabel: icon.label,
          tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, minHeight: 56 },
          tabBarIcon: ({ focused, color, size }) => (
            <Ionicons color={color} name={focused ? icon.on : icon.off} size={size} />
          )
        };
      }}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        component={MessagesScreen}
        name="Messages"
        options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined, tabBarTestID: "tab-messages" }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarTestID: "tab-notifications"
        }}
      />
      <Tabs.Screen component={MeScreen} name="Me" options={{ tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const { colors } = useTheme();
  useRealtime(true);

  return (
    <MainStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.text,
        headerShadowVisible: false,
        headerBackTitle: "",
        contentStyle: { backgroundColor: colors.background }
      }}
    >
      <MainStack.Screen component={MainTabs} name="Tabs" options={{ headerShown: false }} />
      <MainStack.Screen component={ChatScreen} name="Chat" />
      <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "" }} />
      <MainStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
      <MainStack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
      <MainStack.Screen
        component={PreferencesScreen}
        name="Preferences"
        options={{ title: "Préférences de rencontre" }}
      />
      <MainStack.Screen component={PrivacyScreen} name="Privacy" options={{ title: "Confidentialité" }} />
      <MainStack.Screen component={SettingsScreen} name="Settings" options={{ title: "Compte et sécurité" }} />
    </MainStack.Navigator>
  );
}

/** Decides between onboarding and the app once, so finishing a step never tears the wizard down. */
function SignedInNavigator() {
  const profile = useProfile();
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    if (onboarding === null && profile.data !== undefined) {
      setOnboarding(profile.data === null || !profile.data.isComplete);
    }
  }, [onboarding, profile.data]);

  if (profile.isError) {
    return <ErrorView message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  }
  if (onboarding === null) {
    return <LoadingView />;
  }
  if (onboarding) {
    return <OnboardingScreen onFinish={() => setOnboarding(false)} profile={profile.data ?? null} />;
  }
  return <MainNavigator />;
}

function AuthNavigator() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.text,
        headerShadowVisible: false,
        headerBackTitle: "",
        contentStyle: { backgroundColor: colors.background }
      }}
    >
      <AuthStack.Screen component={AuthScreen} name="Auth" options={{ headerShown: false }} />
      <AuthStack.Screen component={ForgotScreen} name="Forgot" options={{ title: "" }} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { colors, isDark } = useTheme();
  const { status, user } = useSession();

  const base = isDark ? DarkTheme : DefaultTheme;
  const theme: Theme = {
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

  let content: React.ReactNode;
  if (status === "booting") {
    content = <LoadingView />;
  } else if (status === "signedOut") {
    content = <AuthNavigator />;
  } else {
    content = <SignedInNavigator key={user?.id} />;
  }

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
