import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React, { createContext, useContext, useEffect, useMemo, useState } from "react";
import ForgotPasswordScreen from "../features/auth/ForgotPasswordScreen";
import LoginScreen from "../features/auth/LoginScreen";
import RegisterScreen from "../features/auth/RegisterScreen";
import WelcomeScreen from "../features/auth/WelcomeScreen";
import ChatScreen from "../features/chat/ChatScreen";
import MatchesScreen from "../features/chat/MatchesScreen";
import DiscoverScreen from "../features/discover/DiscoverScreen";
import NotificationsScreen from "../features/notifications/NotificationsScreen";
import OnboardingScreen from "../features/onboarding/OnboardingScreen";
import EditProfileScreen from "../features/profile/EditProfileScreen";
import MeScreen from "../features/profile/MeScreen";
import MyPreviewScreen from "../features/profile/MyPreviewScreen";
import PreferencesScreen from "../features/profile/PreferencesScreen";
import ProfileDetailScreen from "../features/profile/ProfileDetailScreen";
import BlockedUsersScreen from "../features/settings/BlockedUsersScreen";
import ReportScreen from "../features/settings/ReportScreen";
import SettingsScreen from "../features/settings/SettingsScreen";
import { useMyProfile, useNotifications, useUnreadMessagesCount } from "../state/queries";
import { useSession } from "../state/session";
import { useTheme } from "../theme/ThemeProvider";
import { ErrorState, LoadingState, Screen } from "../ui";
import { describeError } from "../utils/errors";
import type { RootStackParamList, TabParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const OnboardingGate = createContext<{ finish: () => void }>({ finish: () => undefined });
export const useOnboardingGate = () => useContext(OnboardingGate);

const tabIcons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Activity: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

const tabLabels: Record<keyof TabParamList, string> = {
  Discover: "Découvrir",
  Matches: "Messages",
  Activity: "Activité",
  Me: "Profil"
};

function Tabs() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessagesCount();
  const notifications = useNotifications();
  const unreadNotifications = notifications.data?.unreadCount ?? 0;
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabel: tabLabels[route.name],
        tabBarTestID: `tab-${route.name}`,
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons name={tabIcons[route.name][focused ? 0 : 1]} size={size} color={color} />
        )
      })}
    >
      <Tab.Screen name="Discover" component={DiscoverScreen} />
      <Tab.Screen
        name="Matches"
        component={MatchesScreen}
        options={{ tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined }}
      />
      <Tab.Screen
        name="Activity"
        component={NotificationsScreen}
        options={{ tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined }}
      />
      <Tab.Screen name="Me" component={MeScreen} />
    </Tab.Navigator>
  );
}

function SignedInStack() {
  const profile = useMyProfile();
  const [needsOnboarding, setNeedsOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    // Decided once per session so the wizard is not interrupted while the
    // profile fills in step by step.
    if (needsOnboarding === null && profile.data) {
      setNeedsOnboarding(!profile.data.isComplete);
    }
  }, [needsOnboarding, profile.data]);

  const gate = useMemo(() => ({ finish: () => setNeedsOnboarding(false) }), []);

  if (needsOnboarding === null) {
    if (profile.isError) {
      return (
        <Screen>
          <ErrorState message={describeError(profile.error)} onAction={() => void profile.refetch()} />
        </Screen>
      );
    }
    return <LoadingState />;
  }

  return (
    <OnboardingGate.Provider value={gate}>
      {/* The wizard is registered under a different route name in the main
          stack, so finishing onboarding cannot rehydrate the old route. */}
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {needsOnboarding ? (
          <Stack.Screen name="Onboarding" component={OnboardingScreen} />
        ) : (
          <>
            <Stack.Screen name="Tabs" component={Tabs} />
            <Stack.Screen name="Chat" component={ChatScreen} />
            <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ presentation: "modal" }} />
            <Stack.Screen name="MyPreview" component={MyPreviewScreen} options={{ presentation: "modal" }} />
            <Stack.Screen name="EditProfile" component={EditProfileScreen} />
            <Stack.Screen name="Preferences" component={PreferencesScreen} />
            <Stack.Screen name="Settings" component={SettingsScreen} />
            <Stack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
            <Stack.Screen name="Report" component={ReportScreen} options={{ presentation: "modal" }} />
            <Stack.Screen name="CompleteProfile" component={OnboardingScreen} />
          </>
        )}
      </Stack.Navigator>
    </OnboardingGate.Provider>
  );
}

function SignedOutStack() {
  return (
    <Stack.Navigator screenOptions={{ headerShown: false }}>
      <Stack.Screen name="Welcome" component={WelcomeScreen} />
      <Stack.Screen name="Login" component={LoginScreen} />
      <Stack.Screen name="Register" component={RegisterScreen} />
      <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </Stack.Navigator>
  );
}

export function RootNavigator() {
  const { status, userId } = useSession();
  const { colors, dark } = useTheme();
  const navTheme = useMemo<NavTheme>(() => {
    const base = dark ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        background: colors.background,
        card: colors.surface,
        text: colors.text,
        border: colors.border,
        primary: colors.primary,
        notification: colors.primary
      }
    };
  }, [colors, dark]);

  return (
    <NavigationContainer theme={navTheme}>
      {status === "booting" ? (
        <LoadingState />
      ) : status === "signedIn" ? (
        <SignedInStack key={userId ?? "user"} />
      ) : (
        <SignedOutStack />
      )}
    </NavigationContainer>
  );
}
