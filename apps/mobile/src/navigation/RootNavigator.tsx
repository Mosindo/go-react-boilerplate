import React, { useMemo } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { MatchCelebrationProvider } from "../components/MatchCelebration";
import { useUnreadCounts } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { RealtimeProvider } from "../hooks/useRealtime";
import { ForgotPasswordScreen } from "../screens/auth/ForgotPasswordScreen";
import { LoginScreen } from "../screens/auth/LoginScreen";
import { RegisterScreen } from "../screens/auth/RegisterScreen";
import { WelcomeScreen } from "../screens/auth/WelcomeScreen";
import { ChatScreen } from "../screens/chat/ChatScreen";
import { MessagesScreen } from "../screens/chat/MessagesScreen";
import { DiscoverScreen } from "../screens/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../screens/discover/ProfileDetailScreen";
import { LikesScreen } from "../screens/likes/LikesScreen";
import { NotificationsScreen } from "../screens/notifications/NotificationsScreen";
import { OnboardingScreen } from "../screens/onboarding/OnboardingScreen";
import { BlockedScreen } from "../screens/profile/BlockedScreen";
import { ChangePasswordScreen } from "../screens/profile/ChangePasswordScreen";
import { EditProfileScreen } from "../screens/profile/EditProfileScreen";
import { MeScreen } from "../screens/profile/MeScreen";
import { PhotosScreen } from "../screens/profile/PhotosScreen";
import { PreferencesScreen } from "../screens/profile/PreferencesScreen";
import { SettingsScreen } from "../screens/profile/SettingsScreen";
import { useTheme } from "../theme";
import { EmptyState, Icon, LoadingState, Screen, type IconName } from "../ui";
import { navigationRef } from "./navigationRef";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const tabIcons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Likes: ["heart", "heart-outline"],
  Messages: ["chatbubbles", "chatbubbles-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const unread = useUnreadCounts();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, height: 64, paddingTop: 6, paddingBottom: 8 },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarIcon: ({ focused, color, size }) => <Icon color={color} name={tabIcons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen component={LikesScreen} name="Likes" options={{ title: "Likes", tabBarTestID: "tab-likes" }} />
      <Tabs.Screen
        component={MessagesScreen}
        name="Messages"
        options={{
          title: "Messages",
          tabBarTestID: "tab-messages",
          tabBarBadge: unread.conversations > 0 ? unread.conversations : undefined,
          tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary }
        }}
      />
      <Tabs.Screen component={MeScreen} name="Me" options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function AppScreens({ userId }: { userId: string }) {
  return (
    <RealtimeProvider userId={userId}>
      <MatchCelebrationProvider>
        <AppStack.Navigator screenOptions={{ headerShown: false }}>
          <AppStack.Screen component={MainTabs} name="Tabs" />
          <AppStack.Screen component={ChatScreen} name="Chat" />
          <AppStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ presentation: "modal" }} />
          <AppStack.Screen component={EditProfileScreen} name="EditProfile" />
          <AppStack.Screen component={PhotosScreen} name="Photos" />
          <AppStack.Screen component={PreferencesScreen} name="Preferences" />
          <AppStack.Screen component={SettingsScreen} name="Settings" />
          <AppStack.Screen component={NotificationsScreen} name="Notifications" />
          <AppStack.Screen component={BlockedScreen} name="Blocked" />
          <AppStack.Screen component={ChangePasswordScreen} name="ChangePassword" />
        </AppStack.Navigator>
      </MatchCelebrationProvider>
    </RealtimeProvider>
  );
}

function AuthScreens() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" />
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const theme = useTheme();
  const { status, user, retry, signOut } = useAuth();

  const navTheme = useMemo<NavTheme>(() => {
    const base = theme.scheme === "dark" ? DarkTheme : DefaultTheme;
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

  let content: React.ReactNode;
  if (status === "booting") {
    content = (
      <Screen>
        <LoadingState />
      </Screen>
    );
  } else if (status === "unavailable") {
    content = (
      <Screen>
        <EmptyState
          actionLabel="Réessayer"
          icon="cloud-offline-outline"
          message="Impossible de joindre Lueur. Vérifiez votre connexion."
          onAction={retry}
          title="Hors ligne"
        />
        <EmptyState actionLabel="Se déconnecter" icon="log-out-outline" onAction={() => void signOut()} title="" />
      </Screen>
    );
  } else if (status === "signedOut") {
    content = <AuthScreens />;
  } else if (status === "onboarding") {
    content = <OnboardingScreen />;
  } else {
    content = <AppScreens userId={user?.id ?? ""} />;
  }

  return (
    <NavigationContainer ref={navigationRef} theme={navTheme}>
      {content}
    </NavigationContainer>
  );
}
