import React, { useEffect, useMemo, useState } from "react";
import { View } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { ForgotPasswordScreen, ResetPasswordScreen } from "../features/auth/PasswordResetScreens";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { DiscoverScreen } from "../features/discovery/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discovery/ProfileDetailScreen";
import { ConversationsScreen } from "../features/chat/ConversationsScreen";
import { ConversationScreen } from "../features/chat/ConversationScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import {
  BlockedUsersScreen,
  DeleteAccountScreen,
  PhotosScreen,
  PreferencesScreen,
  PrivacyScreen
} from "../features/profile/SettingsScreens";
import { useAuth } from "../state/AuthProvider";
import { useConversations, useMyProfile, useNotifications } from "../state/queries";
import { useTheme } from "../theme/ThemeProvider";
import { ErrorState, Icon, LoadingState, type IconName } from "../ui";
import { errorMessage } from "../api/client";
import { navigationRef } from "./navigationRef";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const tabIcons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  const unreadNotifications = notifications.data?.unreadCount ?? 0;
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ focused, color, size }) => <Icon color={color} name={tabIcons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir" }} />
      <Tabs.Screen
        component={ConversationsScreen}
        name="Matches"
        options={{ title: "Matchs", tabBarBadge: unreadMessages ? (unreadMessages > 9 ? "9+" : unreadMessages) : undefined }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ title: "Activité", tabBarBadge: unreadNotifications ? (unreadNotifications > 9 ? "9+" : unreadNotifications) : undefined }}
      />
      <Tabs.Screen component={MeScreen} name="Me" options={{ title: "Profil" }} />
    </Tabs.Navigator>
  );
}

/** Themed full-screen container for states rendered outside a Screen. */
function Backdrop({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  return <View style={{ flex: 1, backgroundColor: colors.background }}>{children}</View>;
}

function SignedIn() {
  const { data, isError, error, refetch } = useMyProfile();
  // Decided once from the server state, then only ended by the wizard, so
  // uploading the first photo mid-onboarding does not jump to the app.
  const [onboarding, setOnboarding] = useState<boolean | null>(null);
  useEffect(() => {
    if (data && onboarding === null) setOnboarding(!data.complete);
  }, [data, onboarding]);

  if (isError && !data) {
    return (
      <Backdrop>
        <ErrorState message={errorMessage(error)} onRetry={refetch} />
      </Backdrop>
    );
  }
  if (onboarding === null) {
    return (
      <Backdrop>
        <LoadingState />
      </Backdrop>
    );
  }
  if (onboarding) return <OnboardingScreen onFinished={() => setOnboarding(false)} />;

  return (
    <MainStack.Navigator screenOptions={{ headerShown: false }}>
      <MainStack.Screen component={MainTabs} name="Tabs" />
      <MainStack.Screen component={ConversationScreen} name="Conversation" />
      <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ presentation: "modal" }} />
      <MainStack.Screen component={EditProfileScreen} name="EditProfile" />
      <MainStack.Screen component={PhotosScreen} name="Photos" />
      <MainStack.Screen component={PreferencesScreen} name="Preferences" />
      <MainStack.Screen component={PrivacyScreen} name="Privacy" />
      <MainStack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
      <MainStack.Screen component={DeleteAccountScreen} name="DeleteAccount" />
    </MainStack.Navigator>
  );
}

function SignedOut() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" />
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
      <AuthStack.Screen component={ResetPasswordScreen} name="ResetPassword" />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, user, retry } = useAuth();
  const { colors, scheme } = useTheme();
  const navTheme = useMemo<Theme>(() => {
    const base = scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: { ...base.colors, background: colors.background, card: colors.surface, border: colors.border, primary: colors.primary, text: colors.text, notification: colors.primary }
    };
  }, [colors, scheme]);

  return (
    <NavigationContainer ref={navigationRef} theme={navTheme} documentTitle={{ formatter: () => "Lueur" }}>
      {status === "booting" ? (
        <Backdrop>
          <LoadingState />
        </Backdrop>
      ) : status === "unreachable" ? (
        <Backdrop>
          <ErrorState message="Impossible de joindre Lueur. Vérifiez votre connexion." onRetry={retry} />
        </Backdrop>
      ) : status === "signedIn" ? (
        <SignedIn key={user?.id} />
      ) : (
        <SignedOut />
      )}
    </NavigationContainer>
  );
}
