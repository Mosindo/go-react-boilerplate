import React, { useEffect, useMemo, useState } from "react";
import { DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { useMatches, useMyProfile, useNotifications } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { useRealtime } from "../hooks/useRealtime";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import ResetPasswordScreen from "../screens/auth/ResetPasswordScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import ProfileDetailScreen from "../screens/discover/ProfileDetailScreen";
import ConversationScreen from "../screens/matches/ConversationScreen";
import MatchesScreen from "../screens/matches/MatchesScreen";
import ActivityScreen from "../screens/notifications/ActivityScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import BlockedUsersScreen from "../screens/profile/BlockedUsersScreen";
import ChangePasswordScreen from "../screens/profile/ChangePasswordScreen";
import DeleteAccountScreen from "../screens/profile/DeleteAccountScreen";
import EditPhotosScreen from "../screens/profile/EditPhotosScreen";
import EditProfileScreen from "../screens/profile/EditProfileScreen";
import MeScreen from "../screens/profile/MeScreen";
import PreferencesScreen from "../screens/profile/PreferencesScreen";
import SettingsScreen from "../screens/profile/SettingsScreen";
import { colors, typography } from "../theme/tokens";
import { ErrorState, LoadingState, Screen } from "../ui";
import { errorMessage } from "../utils/errors";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const navTheme: Theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: colors.accent,
    background: colors.background,
    card: colors.surface,
    text: colors.text,
    border: colors.border,
    notification: colors.accent
  }
};

const stackOptions = { headerShown: false, contentStyle: { backgroundColor: colors.background } } as const;

function TabsNavigator() {
  const matches = useMatches();
  const notifications = useNotifications();
  const unreadMessages = useMemo(() => matches.data?.pages.flatMap((p) => p.matches).reduce((n, m) => n + m.unreadCount, 0) ?? 0, [matches.data]);
  const unreadActivity = notifications.data?.pages[0]?.unreadCount ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, height: 64, paddingTop: 6, paddingBottom: 8 },
        tabBarLabelStyle: { ...typography.caption, fontSize: 11 },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.textOnAccent },
        tabBarIcon: ({ color, focused, size }) => {
          const icons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
            Discover: ["flame", "flame-outline"],
            Matches: ["chatbubbles", "chatbubbles-outline"],
            Activity: ["notifications", "notifications-outline"],
            Me: ["person-circle", "person-circle-outline"]
          };
          const [on, off] = icons[route.name];
          return <Ionicons name={focused ? on : off} size={size} color={color} />;
        }
      })}
      sceneContainerStyle={{ backgroundColor: colors.background }}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        name="Matches"
        component={MatchesScreen}
        options={{ title: "Matchs", tabBarBadge: unreadMessages || undefined, tabBarTestID: "tab-matches" }}
      />
      <Tabs.Screen
        name="Activity"
        component={ActivityScreen}
        options={{ title: "Activité", tabBarBadge: unreadActivity || undefined, tabBarTestID: "tab-activity" }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function MainApp() {
  useRealtime(true);
  return (
    <AppStack.Navigator screenOptions={stackOptions}>
      <AppStack.Screen name="Tabs" component={TabsNavigator} />
      <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ presentation: "modal" }} />
      <AppStack.Screen name="Conversation" component={ConversationScreen} />
      <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
      <AppStack.Screen name="EditPhotos" component={EditPhotosScreen} />
      <AppStack.Screen name="Preferences" component={PreferencesScreen} />
      <AppStack.Screen name="Settings" component={SettingsScreen} />
      <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
      <AppStack.Screen name="ChangePassword" component={ChangePasswordScreen} />
      <AppStack.Screen name="DeleteAccount" component={DeleteAccountScreen} />
    </AppStack.Navigator>
  );
}

/**
 * Signed-in gate: users with an incomplete profile go through onboarding
 * first. The decision is taken once per session so the onboarding flow is not
 * interrupted when the profile becomes "complete" halfway through it.
 */
function SignedInGate() {
  const me = useMyProfile();
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    if (onboarding === null && me.data) {
      setOnboarding(!me.data.completeness.isComplete);
    }
  }, [me.data, onboarding]);

  if (!me.data || onboarding === null) {
    return <Screen>{me.error ? <ErrorState message={errorMessage(me.error)} onRetry={() => me.refetch()} /> : <LoadingState />}</Screen>;
  }
  if (onboarding) {
    return <OnboardingScreen me={me.data} onFinish={() => setOnboarding(false)} />;
  }
  return <MainApp />;
}

export function RootNavigator() {
  const { status } = useAuth();
  return (
    <NavigationContainer theme={navTheme} documentTitle={{ formatter: () => "Lueur" }}>
      {status === "booting" ? (
        <Screen>
          <LoadingState />
        </Screen>
      ) : status === "signedOut" ? (
        <AuthStack.Navigator screenOptions={stackOptions}>
          <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
          <AuthStack.Screen name="Login" component={LoginScreen} />
          <AuthStack.Screen name="Register" component={RegisterScreen} />
          <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
          <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
        </AuthStack.Navigator>
      ) : (
        <SignedInGate />
      )}
    </NavigationContainer>
  );
}
