import React, { useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { SignInScreen } from "../features/auth/SignInScreen";
import { DiscoverScreen } from "../features/discovery/DiscoverScreen";
import { MatchCelebration } from "../features/discovery/MatchCelebration";
import { ChatScreen } from "../features/matches/ChatScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { ProfileDetailScreen } from "../features/profile/ProfileDetailScreen";
import { AccountScreen } from "../features/settings/AccountScreen";
import { PreferencesScreen } from "../features/settings/PreferencesScreen";
import { PrivacyScreen } from "../features/settings/PrivacyScreen";
import { useAuth } from "../hooks/useAuth";
import { keys, useMatches, useMyProfile, useUnreadCount } from "../hooks/queries";
import { useLocationRefresh } from "../hooks/useLocation";
import { useRealtime } from "../hooks/useRealtime";
import { useTheme } from "../theme/ThemeProvider";
import { ErrorState, LoadingState, Screen } from "../ui";
import type { AuthStackParamList, MainStackParamList, MainTabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

type IconName = React.ComponentProps<typeof Ionicons>["name"];
const tabIcons: Record<keyof MainTabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Activity: ["notifications", "notifications-outline"],
  Me: ["person", "person-outline"]
};
const tabLabels: Record<keyof MainTabParamList, string> = { Discover: "Découvrir", Matches: "Messages", Activity: "Activité", Me: "Profil" };

function MainTabs() {
  const theme = useTheme();
  const unread = useUnreadCount();
  const matches = useMatches();
  const unreadMessages = matches.data?.pages[0]?.items.reduce((n, m) => n + m.unreadCount, 0) ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: theme.colors.primary,
        tabBarInactiveTintColor: theme.colors.textSubtle,
        tabBarStyle: { backgroundColor: theme.colors.surface, borderTopColor: theme.colors.border },
        tabBarLabel: tabLabels[route.name],
        tabBarIcon: ({ focused, color, size }) => <Ionicons color={color} name={tabIcons[route.name][focused ? 0 : 1]} size={size} />,
        tabBarBadgeStyle: { backgroundColor: theme.colors.primary, color: theme.colors.onPrimary }
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" />
      <Tabs.Screen component={MatchesScreen} name="Matches" options={{ tabBarBadge: unreadMessages || undefined }} />
      <Tabs.Screen component={NotificationsScreen} name="Activity" options={{ tabBarBadge: unread.data || undefined }} />
      <Tabs.Screen component={MeScreen} name="Me" />
    </Tabs.Navigator>
  );
}

function SignedIn() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const me = useMyProfile();
  const ready = !!me.data && !needsOnboarding(me.data.completion.missing);
  useRealtime(ready, user?.id ?? null);
  const onLocationUpdated = useCallback(() => void qc.invalidateQueries({ queryKey: keys.me }), [qc]);
  useLocationRefresh(ready && !!me.data?.profile?.hasLocation, onLocationUpdated);

  if (me.isLoading) return <LoadingState label="Chargement de votre profil…" />;
  if (me.error || !me.data)
    return (
      <Screen>
        <ErrorState onRetry={() => me.refetch()} />
      </Screen>
    );
  if (needsOnboarding(me.data.completion.missing)) return <OnboardingScreen me={me.data} />;

  return (
    <>
      <MainStack.Navigator screenOptions={{ headerShown: false }}>
        <MainStack.Screen component={MainTabs} name="Tabs" />
        <MainStack.Screen component={ChatScreen} name="Chat" />
        <MainStack.Screen component={ProfileDetailScreen} name="ProfileDetail" />
        <MainStack.Screen component={EditProfileScreen} name="EditProfile" />
        <MainStack.Screen component={PreferencesScreen} name="Preferences" />
        <MainStack.Screen component={PrivacyScreen} name="Privacy" />
        <MainStack.Screen component={AccountScreen} name="Account" />
      </MainStack.Navigator>
      <MatchCelebration />
    </>
  );
}

function needsOnboarding(missing: string[]) {
  return missing.includes("profile") || missing.includes("preferences") || missing.includes("photos");
}

export function RootNavigator() {
  const theme = useTheme();
  const { status } = useAuth();
  const navTheme = useMemo(() => {
    const base = theme.scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        background: theme.colors.background,
        card: theme.colors.surface,
        text: theme.colors.text,
        border: theme.colors.border,
        primary: theme.colors.primary
      }
    };
  }, [theme]);

  if (status === "booting") return <LoadingState />;

  return (
    <NavigationContainer theme={navTheme}>
      {status === "signedIn" ? (
        <SignedIn />
      ) : (
        <AuthStack.Navigator screenOptions={{ headerShown: false }}>
          <AuthStack.Screen component={SignInScreen} name="SignIn" />
          <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        </AuthStack.Navigator>
      )}
    </NavigationContainer>
  );
}
