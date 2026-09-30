import React, { useEffect, useMemo, useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { ErrorState, IconButton, LoadingState } from "../design/components";
import { useTheme } from "../design/ThemeProvider";
import ForgotPasswordScreen from "../features/auth/ForgotPasswordScreen";
import SignInScreen from "../features/auth/SignInScreen";
import SignUpScreen from "../features/auth/SignUpScreen";
import WelcomeScreen from "../features/auth/WelcomeScreen";
import ChatScreen from "../features/chat/ChatScreen";
import ConversationsScreen from "../features/chat/ConversationsScreen";
import { useUnreadMessagesCount } from "../features/chat/hooks";
import DiscoverScreen from "../features/discover/DiscoverScreen";
import ProfileDetailScreen from "../features/discover/ProfileDetailScreen";
import { useNotifications } from "../features/notifications/hooks";
import NotificationsScreen from "../features/notifications/NotificationsScreen";
import { OnboardingContext } from "../features/onboarding/OnboardingContext";
import OnboardingScreen from "../features/onboarding/OnboardingScreen";
import BlockedUsersScreen from "../features/profile/BlockedUsersScreen";
import EditProfileScreen from "../features/profile/EditProfileScreen";
import { useProfile } from "../features/profile/hooks";
import MyProfileScreen from "../features/profile/MyProfileScreen";
import ModerationScreen from "../features/moderation/ModerationScreen";
import PreferencesScreen from "../features/profile/PreferencesScreen";
import SettingsScreen from "../features/profile/SettingsScreen";
import { errorMessage } from "../lib/api/client";
import { RealtimeProvider } from "../lib/realtime/RealtimeProvider";
import { useSession } from "../lib/session/SessionProvider";
import type { AppStackParamList, AuthStackParamList, MainTabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

function AuthNavigator() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator screenOptions={{ headerShadowVisible: false, headerTitle: "", headerTintColor: colors.text, headerStyle: { backgroundColor: colors.background } }}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" options={{ headerShown: false }} />
      <AuthStack.Screen component={SignInScreen} name="SignIn" />
      <AuthStack.Screen component={SignUpScreen} name="SignUp" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
    </AuthStack.Navigator>
  );
}

function MainTabs() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessagesCount();
  const { data: notifications } = useNotifications();
  const icons: Record<keyof MainTabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
    Discover: ["flame", "flame-outline"],
    Messages: ["chatbubbles", "chatbubbles-outline"],
    Notifications: ["notifications", "notifications-outline"],
    Me: ["person-circle", "person-circle-outline"]
  };
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ focused, color, size }) => <Ionicons color={color} name={icons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        component={ConversationsScreen}
        name="Messages"
        options={{ title: "Messages", tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined, tabBarTestID: "tab-messages" }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{ title: "Activité", tabBarBadge: notifications?.unreadCount ? notifications.unreadCount : undefined, tabBarTestID: "tab-notifications" }}
      />
      <Tabs.Screen component={MyProfileScreen} name="Me" options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

/** Staff accounts without a dating profile only get the moderation space. */
function ModeratorOnlyNavigator() {
  const { colors } = useTheme();
  const { signOut } = useSession();
  return (
    <AppStack.Navigator screenOptions={{ headerShadowVisible: false, headerTintColor: colors.text, headerStyle: { backgroundColor: colors.background } }}>
      <AppStack.Screen
        component={ModerationScreen}
        name="Moderation"
        options={{
          title: "Modération",
          headerRight: () => <IconButton icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} testID="moderator-signout" />
        }}
      />
    </AppStack.Navigator>
  );
}

function SignedInNavigator({ userId }: { userId: string }) {
  const { colors } = useTheme();
  const { isModerator } = useSession();
  const { data: profile, isLoading, error, refetch } = useProfile();
  // null until the profile is loaded; then fixed for this session so the
  // optional onboarding steps are not skipped once the profile is complete.
  const [onboarding, setOnboarding] = useState<boolean | null>(null);
  useEffect(() => {
    if (profile && onboarding === null) {
      setOnboarding(!profile.completeness.complete);
    }
  }, [onboarding, profile]);
  const onboardingContext = useMemo(() => ({ finish: () => setOnboarding(false) }), []);

  if (isLoading || (profile && onboarding === null)) return <LoadingState />;
  if (error || !profile) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  if (isModerator && !profile.completeness.complete) {
    return <ModeratorOnlyNavigator />;
  }

  if (onboarding || !profile.completeness.complete) {
    return (
      <OnboardingContext.Provider value={onboardingContext}>
        <OnboardingScreen />
      </OnboardingContext.Provider>
    );
  }

  return (
    <RealtimeProvider userId={userId}>
      <AppStack.Navigator
        screenOptions={{
          headerShadowVisible: false,
          headerTintColor: colors.text,
          headerStyle: { backgroundColor: colors.background },
          headerBackTitle: "Retour",
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <AppStack.Screen component={MainTabs} name="Tabs" options={{ headerShown: false }} />
        <AppStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "", headerTransparent: true, headerTintColor: "#fff" }} />
        <AppStack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
        <AppStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
        <AppStack.Screen component={PreferencesScreen} name="Preferences" options={{ title: "Préférences" }} />
        <AppStack.Screen component={SettingsScreen} name="Settings" options={{ title: "Confidentialité et compte" }} />
        <AppStack.Screen component={BlockedUsersScreen} name="BlockedUsers" options={{ title: "Personnes bloquées" }} />
        <AppStack.Screen component={ModerationScreen} name="Moderation" options={{ title: "Modération" }} />
      </AppStack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const theme = useTheme();
  const { status, userId } = useSession();
  const navTheme: NavTheme = useMemo(() => {
    const base = theme.dark ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: theme.colors.primary,
        background: theme.colors.background,
        card: theme.colors.surface,
        text: theme.colors.text,
        border: theme.colors.border,
        notification: theme.colors.primary
      }
    };
  }, [theme]);

  return (
    <NavigationContainer theme={navTheme}>
      {status === "restoring" ? (
        <LoadingState />
      ) : status === "signedIn" && userId ? (
        <SignedInNavigator key={userId} userId={userId} />
      ) : (
        <AuthNavigator />
      )}
    </NavigationContainer>
  );
}
