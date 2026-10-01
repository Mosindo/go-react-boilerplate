import React, { useEffect, useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { useQuery } from "@tanstack/react-query";
import { chatApi, notificationsApi } from "../api/endpoints";
import { keys } from "../api/keys";
import { useAuth } from "../auth/AuthProvider";
import { useFeedback } from "../components/Feedback";
import { ErrorState, IconButton, Loading, Screen } from "../components/ui";
import { useRealtime } from "../realtime/useRealtime";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { ForgotScreen, LoginScreen, RegisterScreen, WelcomeScreen } from "../screens/AuthScreens";
import { MatchesScreen } from "../screens/MatchesScreen";
import { BlockedScreen, EditProfileScreen, MeScreen, PhotosScreen, PreferencesScreen, SettingsScreen, useProfile } from "../screens/MeScreens";
import { NotificationsScreen } from "../screens/NotificationsScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { ProfileDetailScreen } from "../screens/ProfileDetailScreen";
import { ReportScreen } from "../screens/ReportScreen";
import { useTheme } from "../theme/theme";
import type { AuthStackParamList, RootStackParamList, TabParamList } from "./types";

/** Header back button with an accessible name (the default one is an unlabeled icon on web). */
function backButton({ navigation }: { navigation: { goBack: () => void; canGoBack: () => boolean } }) {
  return {
    headerLeft: () => (navigation.canGoBack() ? <IconButton icon="arrow-back" label="Retour" onPress={() => navigation.goBack()} testID="header-back" /> : null)
  };
}

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const Stack = createNativeStackNavigator<RootStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={(props) => ({ headerShadowVisible: false, headerTitle: "", ...backButton(props) })}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="Forgot" component={ForgotScreen} />
    </AuthStack.Navigator>
  );
}

const TAB_ICONS: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Me: ["person", "person-outline"]
};

const TAB_LABELS: Record<keyof TabParamList, string> = {
  Discover: "Découvrir",
  Matches: "Matchs",
  Notifications: "Notifications",
  Me: "Profil"
};

function TabNavigator() {
  const t = useTheme();
  const unread = useQuery({ queryKey: keys.unread, queryFn: notificationsApi.unreadCount, refetchInterval: 60_000 });
  const conversations = useQuery({ queryKey: keys.conversations, queryFn: chatApi.conversations });
  const unreadMessages = (conversations.data ?? []).reduce((n, c) => n + c.unreadCount, 0);
  const badges: Partial<Record<keyof TabParamList, number>> = { Notifications: unread.data ?? 0, Matches: unreadMessages };

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: t.primary,
        tabBarInactiveTintColor: t.textMuted,
        tabBarStyle: { backgroundColor: t.surface, borderTopColor: t.border },
        tabBarLabel: TAB_LABELS[route.name],
        tabBarButtonTestID: `tab-${route.name}`,
        tabBarBadge: badges[route.name] ? badges[route.name] : undefined,
        tabBarBadgeStyle: { backgroundColor: t.primary, color: t.onPrimary },
        tabBarIcon: ({ focused, color, size }) => {
          const [on, off] = TAB_ICONS[route.name];
          return <Ionicons name={focused ? on : off} size={size} color={color} />;
        }
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} />
      <Tabs.Screen name="Matches" component={MatchesScreen} />
      <Tabs.Screen name="Notifications" component={NotificationsScreen} />
      <Tabs.Screen name="Me" component={MeScreen} />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  return (
    <Stack.Navigator screenOptions={(props) => ({ headerShadowVisible: false, ...backButton(props) })}>
      <Stack.Screen name="Tabs" component={TabNavigator} options={{ headerShown: false }} />
      <Stack.Screen name="Chat" component={ChatScreen} />
      <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "" }} />
      <Stack.Screen name="Report" component={ReportScreen} options={{ title: "Signalement", presentation: "modal" }} />
      <Stack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier mon profil" }} />
      <Stack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
      <Stack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Préférences" }} />
      <Stack.Screen name="Settings" component={SettingsScreen} options={{ title: "Confidentialité et compte" }} />
      <Stack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Personnes bloquées" }} />
    </Stack.Navigator>
  );
}

function SignedIn() {
  const { toast } = useFeedback();
  const profile = useProfile();
  // Onboarding is decided once, when the profile first loads, so it is not torn down mid-flow
  // the moment the first photo makes the profile "complete".
  const [onboarding, setOnboarding] = useState<boolean | null>(null);
  useEffect(() => {
    if (profile.data && onboarding === null) setOnboarding(!profile.data.complete);
  }, [profile.data, onboarding]);

  useRealtime(onboarding === false, (name) => toast(`Nouveau match avec ${name} !`, "success"));

  if (profile.error && !profile.data) {
    return <Screen><ErrorState message={(profile.error as Error).message} onRetry={() => void profile.refetch()} /></Screen>;
  }
  if (!profile.data || onboarding === null) return <Screen><Loading /></Screen>;
  if (onboarding) return <OnboardingScreen profile={profile.data} onDone={() => setOnboarding(false)} />;
  return <MainNavigator />;
}

export function RootNavigator() {
  const t = useTheme();
  const { status, bootError, retryBoot } = useAuth();

  const base = t.isDark ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
    ...base,
    colors: { ...base.colors, background: t.background, card: t.background, text: t.text, border: t.border, primary: t.primary, notification: t.primary }
  };

  let content: React.ReactNode;
  if (status === "booting") {
    content = bootError ? (
      <Screen><ErrorState message={bootError} onRetry={retryBoot} /></Screen>
    ) : (
      <Screen><Loading label="Restauration de la session…" /></Screen>
    );
  } else if (status === "signedOut") {
    content = <AuthNavigator />;
  } else {
    content = <SignedIn />;
  }
  return <NavigationContainer theme={navTheme}>{content}</NavigationContainer>;
}
