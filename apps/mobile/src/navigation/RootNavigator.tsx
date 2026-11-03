import React, { useEffect, useState } from "react";
import { Text } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, createNavigationContainerRef, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useColorScheme } from "react-native";
import { ApiError, errorMessage } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { useMatch } from "../features/match/MatchContext";
import { useTheme } from "../theme";
import { ErrorState, Loading } from "../ui/components";
import { ForgotPasswordScreen, LoginScreen, RegisterScreen } from "../screens/AuthScreens";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { LikesScreen } from "../screens/LikesScreen";
import { BlockedScreen, EditProfileScreen, MeScreen, NotificationsScreen, SettingsScreen, useMyProfile } from "../screens/MeScreens";
import { MessagesScreen, useConversations } from "../screens/MessagesScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { UserProfileScreen } from "../screens/UserProfileScreen";
import type { AuthStackParams, MainStackParams, TabParams } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tabs = createBottomTabNavigator<TabParams>();
const navigationRef = createNavigationContainerRef<MainStackParams>();

function TabIcon({ symbol, color }: { symbol: string; color: string }) {
  return <Text style={{ color, fontSize: 20 }}>{symbol}</Text>;
}

function TabsNavigator() {
  const theme = useTheme();
  const conversations = useConversations();
  const unread = (conversations.data ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.textMuted,
        tabBarStyle: { backgroundColor: theme.surface, borderTopColor: theme.border }
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          tabBarIcon: ({ color }) => <TabIcon symbol="🔥" color={color} />,
          tabBarAccessibilityLabel: "Découvrir"
        }}
      />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{ title: "Likes", tabBarIcon: ({ color }) => <TabIcon symbol="♥" color={color} /> }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarBadge: unread > 0 ? unread : undefined,
          tabBarIcon: ({ color }) => <TabIcon symbol="💬" color={color} />
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{ title: "Profil", tabBarIcon: ({ color }) => <TabIcon symbol="☺" color={color} /> }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const { setOpenChat } = useMatch();
  useEffect(() => {
    setOpenChat(
      (info) =>
        navigationRef.isReady() &&
        navigationRef.navigate("Chat", { conversationId: info.conversationId, matchId: info.matchId, user: info.user })
    );
    return () => setOpenChat(null);
  }, [setOpenChat]);
  return (
    <MainStack.Navigator>
      <MainStack.Screen name="Tabs" component={TabsNavigator} options={{ headerShown: false }} />
      <MainStack.Screen name="Chat" component={ChatScreen} />
      <MainStack.Screen name="UserProfile" component={UserProfileScreen} options={{ title: "Profil" }} />
      <MainStack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier mon profil" }} />
      <MainStack.Screen name="Settings" component={SettingsScreen} options={{ title: "Réglages" }} />
      <MainStack.Screen name="Notifications" component={NotificationsScreen} options={{ title: "Notifications" }} />
      <MainStack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Utilisateurs bloqués" }} />
    </MainStack.Navigator>
  );
}

/** Shows onboarding until the profile is complete, then the main app. */
function SignedIn() {
  const profile = useMyProfile();
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    if (onboarding !== null || profile.isLoading) {
      return;
    }
    if (profile.data) {
      setOnboarding(!profile.data.complete);
    } else if (profile.error instanceof ApiError && profile.error.status === 404) {
      setOnboarding(true);
    }
  }, [onboarding, profile.data, profile.error, profile.isLoading]);

  if (onboarding === null) {
    if (profile.isError && !(profile.error instanceof ApiError && profile.error.status === 404)) {
      return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
    }
    return <Loading label="Chargement de votre profil…" />;
  }
  if (onboarding) {
    return <OnboardingScreen profile={profile.data} onDone={() => setOnboarding(false)} />;
  }
  return <MainNavigator />;
}

export function RootNavigator() {
  const { booting, user } = useAuth();
  const scheme = useColorScheme();
  const palette = useTheme();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  const theme: Theme = {
    ...base,
    colors: {
      ...base.colors,
      primary: palette.primary,
      background: palette.background,
      card: palette.surface,
      text: palette.text,
      border: palette.border
    }
  };

  if (booting) {
    return <Loading label="Chargement…" />;
  }
  return (
    <NavigationContainer ref={navigationRef} theme={theme}>
      {user ? (
        <SignedIn key={user.id} />
      ) : (
        <AuthStack.Navigator screenOptions={{ headerShown: false }}>
          <AuthStack.Screen name="Login" component={LoginScreen} />
          <AuthStack.Screen name="Register" component={RegisterScreen} />
          <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} options={{ headerShown: true, title: "" }} />
        </AuthStack.Navigator>
      )}
    </NavigationContainer>
  );
}
