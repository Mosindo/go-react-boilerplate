import React, { useEffect, useState } from "react";
import { Platform } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useChats, useMyProfile, useNotifications } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { RealtimeProvider, useRealtime } from "../hooks/useRealtime";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import ActivityScreen from "../screens/main/ActivityScreen";
import ChatScreen from "../screens/main/ChatScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import MatchesScreen from "../screens/main/MatchesScreen";
import MeScreen from "../screens/main/MeScreen";
import ProfileDetailScreen from "../screens/main/ProfileDetailScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import AccountScreen from "../screens/settings/AccountScreen";
import EditProfileScreen from "../screens/settings/EditProfileScreen";
import PhotosScreen from "../screens/settings/PhotosScreen";
import PreferencesScreen from "../screens/settings/PreferencesScreen";
import PrivacyScreen from "../screens/settings/PrivacyScreen";
import { useTheme } from "../theme";
import { ErrorState, Icon, LoadingState, Screen, type IconName } from "../ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const tabIcons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Activity: ["notifications", "notifications-outline"],
  Me: ["person", "person-outline"]
};

function Tabs() {
  const { colors } = useTheme();
  const { connected } = useRealtime();
  const chats = useChats(connected ? false : 15_000);
  const notifications = useNotifications(connected ? false : 20_000);
  const unreadChats = (chats.data ?? []).filter((c) => c.unreadCount > 0).length;
  const unreadNotifications = notifications.data?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.border,
          height: Platform.OS === "web" ? 64 : undefined
        },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.textOnAccent },
        tabBarIcon: ({ focused }) => (
          <Icon color={focused ? "accent" : "textSubtle"} name={tabIcons[route.name][focused ? 0 : 1]} size={24} />
        )
      })}
    >
      <Tab.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        component={MatchesScreen}
        name="Matches"
        options={{ title: "Matchs", tabBarBadge: unreadChats > 0 ? unreadChats : undefined, tabBarTestID: "tab-matches" }}
      />
      <Tab.Screen
        component={ActivityScreen}
        name="Activity"
        options={{
          title: "Activité",
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarTestID: "tab-activity"
        }}
      />
      <Tab.Screen component={MeScreen} name="Me" options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tab.Navigator>
  );
}

function SignedIn({ userId }: { userId: string }) {
  const profile = useMyProfile();
  const [onboarding, setOnboarding] = useState(false);

  // Once onboarding starts it stays on screen until the member finishes the
  // optional steps, even though the profile becomes "complete" after photos.
  useEffect(() => {
    if (profile.data && !profile.data.isComplete) {
      setOnboarding(true);
    }
  }, [profile.data]);

  if (profile.isLoading) {
    return (
      <Screen>
        <LoadingState label="Chargement de votre profil…" />
      </Screen>
    );
  }
  if (!profile.data) {
    return (
      <Screen>
        <ErrorState message="Impossible de charger votre profil." onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  if (onboarding || !profile.data.isComplete) {
    return <OnboardingScreen onDone={() => setOnboarding(false)} profile={profile.data} />;
  }

  return (
    <RealtimeProvider userId={userId}>
      <AppStack.Navigator screenOptions={{ headerShown: false }}>
        <AppStack.Screen component={Tabs} name="Tabs" />
        <AppStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ presentation: "modal" }} />
        <AppStack.Screen component={ChatScreen} name="Chat" />
        <AppStack.Screen component={EditProfileScreen} name="EditProfile" />
        <AppStack.Screen component={PhotosScreen} name="Photos" />
        <AppStack.Screen component={PreferencesScreen} name="Preferences" />
        <AppStack.Screen component={PrivacyScreen} name="Privacy" />
        <AppStack.Screen component={AccountScreen} name="Account" />
      </AppStack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status, user } = useAuth();
  const { colors, dark } = useTheme();
  const base = dark ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.accent,
      text: colors.text,
      notification: colors.accent
    }
  };

  return (
    <NavigationContainer documentTitle={{ formatter: () => "Lueur" }} theme={navTheme}>
      {status === "booting" ? (
        <Screen>
          <LoadingState />
        </Screen>
      ) : status === "signedIn" && user ? (
        <SignedIn key={user.id} userId={user.id} />
      ) : (
        <AuthStack.Navigator screenOptions={{ headerShown: false }}>
          <AuthStack.Screen component={WelcomeScreen} name="Welcome" />
          <AuthStack.Screen component={LoginScreen} name="Login" />
          <AuthStack.Screen component={RegisterScreen} name="Register" />
          <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
        </AuthStack.Navigator>
      )}
    </NavigationContainer>
  );
}
