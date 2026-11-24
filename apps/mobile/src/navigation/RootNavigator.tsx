import React, { useMemo } from "react";
import { View } from "react-native";
import { DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useAuth } from "../auth/AuthProvider";
import { MatchCelebration } from "../components/discover/MatchCelebration";
import { ErrorState, LoadingState } from "../components/States";
import { useConversations, useMyProfile, useUnreadNotifications } from "../hooks/queries";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import SignInScreen from "../screens/auth/SignInScreen";
import SignUpScreen from "../screens/auth/SignUpScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import ProfileDetailScreen from "../screens/discover/ProfileDetailScreen";
import ChatScreen from "../screens/matches/ChatScreen";
import MatchesScreen from "../screens/matches/MatchesScreen";
import ActivityScreen from "../screens/notifications/ActivityScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import AccountScreen from "../screens/profile/AccountScreen";
import EditPhotosScreen from "../screens/profile/EditPhotosScreen";
import EditProfileScreen from "../screens/profile/EditProfileScreen";
import MeScreen from "../screens/profile/MeScreen";
import PreferencesScreen from "../screens/profile/PreferencesScreen";
import PrivacyScreen from "../screens/profile/PrivacyScreen";
import { colors } from "../theme/tokens";
import { errorMessage } from "../utils/errors";
import { navigationRef } from "./navigationRef";
import { OnboardingHoldProvider, useOnboardingHold } from "./onboardingHold";
import type { RootStackParamList, TabParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const theme: Theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: colors.primary,
    background: colors.background,
    card: colors.surface,
    text: colors.text,
    border: colors.border,
    notification: colors.primary
  }
};

const tabIcons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Activity: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const conversations = useConversations();
  const unreadNotifications = useUnreadNotifications();
  const unreadMessages = useMemo(
    () => (conversations.data?.pages ?? []).flatMap((p) => p.conversations).reduce((sum, c) => sum + c.unreadCount, 0),
    [conversations.data]
  );
  const badge = (n: number | undefined) => (n && n > 0 ? (n > 9 ? "9+" : n) : undefined);

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ focused, color, size }) => <Ionicons name={tabIcons[route.name][focused ? 0 : 1]} color={color} size={size} />,
        tabBarButtonTestID: `tab-${route.name}`
      })}
    >
      <Tab.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir" }} />
      <Tab.Screen name="Matches" component={MatchesScreen} options={{ title: "Matchs", tabBarBadge: badge(unreadMessages) }} />
      <Tab.Screen name="Activity" component={ActivityScreen} options={{ title: "Activité", tabBarBadge: badge(unreadNotifications.data) }} />
      <Tab.Screen name="Me" component={MeScreen} options={{ title: "Profil" }} />
    </Tab.Navigator>
  );
}

function SignedInStack() {
  const profile = useMyProfile();
  const { hold } = useOnboardingHold();

  if (profile.isPending) return <LoadingState label="Chargement de ton profil…" />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;

  const showOnboarding = !profile.data.isComplete || hold;
  return (
    <Stack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
      {showOnboarding ? (
        <Stack.Screen name="Onboarding" component={OnboardingScreen} />
      ) : (
        <>
          <Stack.Screen name="Main" component={MainTabs} />
          <Stack.Screen name="Chat" component={ChatScreen} />
          <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ presentation: "modal" }} />
          <Stack.Screen name="EditProfile" component={EditProfileScreen} />
          <Stack.Screen name="EditPhotos" component={EditPhotosScreen} />
          <Stack.Screen name="Preferences" component={PreferencesScreen} />
          <Stack.Screen name="Privacy" component={PrivacyScreen} />
          <Stack.Screen name="Account" component={AccountScreen} />
        </>
      )}
    </Stack.Navigator>
  );
}

function SignedOutStack() {
  return (
    <Stack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
      <Stack.Screen name="Welcome" component={WelcomeScreen} />
      <Stack.Screen name="SignIn" component={SignInScreen} />
      <Stack.Screen name="SignUp" component={SignUpScreen} />
      <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </Stack.Navigator>
  );
}

export function RootNavigator() {
  const { status, account } = useAuth();
  return (
    <NavigationContainer ref={navigationRef} theme={theme} documentTitle={{ formatter: () => "Lueur" }}>
      {status === "booting" ? (
        <View style={{ flex: 1, backgroundColor: colors.background }}>
          <LoadingState />
        </View>
      ) : status === "signedIn" && account ? (
        <RealtimeProvider userId={account.id}>
          <OnboardingHoldProvider>
            <SignedInStack />
            <MatchCelebration />
          </OnboardingHoldProvider>
        </RealtimeProvider>
      ) : (
        <SignedOutStack />
      )}
    </NavigationContainer>
  );
}
