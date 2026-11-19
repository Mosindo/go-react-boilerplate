import React, { useCallback, useMemo, useState } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, createNavigationContainerRef } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { useQuery } from "@tanstack/react-query";
import { discoveryApi } from "../api/endpoints";
import { queryKeys } from "../api/queryKeys";
import { errorMessage } from "../api/client";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { ChatScreen } from "../features/chat/ChatScreen";
import { ConversationsScreen, useConversations } from "../features/chat/ConversationsScreen";
import { DiscoverScreen } from "../features/discovery/DiscoverScreen";
import { MatchModal } from "../features/discovery/MatchModal";
import { ProfileDetailScreen } from "../features/discovery/ProfileDetailScreen";
import { LikesScreen } from "../features/likes/LikesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { BlockedUsersScreen } from "../features/profile/BlockedUsersScreen";
import { DeleteAccountScreen } from "../features/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { MyProfileScreen } from "../features/profile/MyProfileScreen";
import { PhotosScreen } from "../features/profile/PhotosScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { SettingsScreen } from "../features/profile/SettingsScreen";
import { useMatchCelebration } from "../state/matchCelebration";
import { useRealtime } from "../state/realtime";
import { useSession } from "../state/session";
import { useTheme } from "../theme/ThemeProvider";
import { Button, ErrorState, Icon, LoadingState, Screen, type IconName } from "../ui";
import type { AppStackParamList, AuthStackParamList, MainTabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

export const navigationRef = createNavigationContainerRef<AppStackParamList>();

const tabIcons: Record<keyof MainTabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Likes: ["heart", "heart-outline"],
  Chats: ["chatbubbles", "chatbubbles-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const likes = useQuery({ queryKey: queryKeys.likesReceived, queryFn: discoveryApi.likesReceived });
  const unreadThreads = conversations.data?.pages.flatMap((p) => p.conversations).filter((c) => c.unreadCount > 0).length ?? 0;
  const likeCount = likes.data?.length ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: "#1A0F15" },
        tabBarIcon: ({ focused, color, size }) => <Icon name={tabIcons[route.name][focused ? 0 : 1]} color={color} size={size} />
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{ title: "Likes", tabBarBadge: likeCount > 0 ? likeCount : undefined, tabBarTestID: "tab-likes" }}
      />
      <Tabs.Screen
        name="Chats"
        component={ConversationsScreen}
        options={{ title: "Messages", tabBarBadge: unreadThreads > 0 ? unreadThreads : undefined, tabBarTestID: "tab-chats" }}
      />
      <Tabs.Screen name="Me" component={MyProfileScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function AppScreens() {
  useRealtime();
  const { current, dismiss } = useMatchCelebration();
  return (
    <>
      <AppStack.Navigator screenOptions={{ headerShown: false }}>
        <AppStack.Screen name="Tabs" component={MainTabs} />
        <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ presentation: "modal" }} />
        <AppStack.Screen name="Chat" component={ChatScreen} />
        <AppStack.Screen name="Notifications" component={NotificationsScreen} />
        <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
        <AppStack.Screen name="Photos" component={PhotosScreen} />
        <AppStack.Screen name="Preferences" component={PreferencesScreen} />
        <AppStack.Screen name="Settings" component={SettingsScreen} />
        <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
        <AppStack.Screen name="DeleteAccount" component={DeleteAccountScreen} />
      </AppStack.Navigator>
      <MatchModal
        match={current}
        onClose={dismiss}
        onMessage={(match) => {
          dismiss();
          if (navigationRef.isReady()) {
            navigationRef.navigate("Chat", { conversationId: match.conversationId, name: match.user.firstName });
          }
        }}
      />
    </>
  );
}

function AuthScreens() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { colors, scheme } = useTheme();
  const { status, profile, profileError, refetchProfile, signOut } = useSession();
  const [onboarding, setOnboarding] = useState(false);
  const startOnboarding = useCallback(() => setOnboarding(true), []);
  const finishOnboarding = useCallback(() => setOnboarding(false), []);

  const navTheme = useMemo(() => {
    const base = scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        background: colors.background,
        card: colors.surface,
        text: colors.text,
        border: colors.border,
        primary: colors.primary
      }
    };
  }, [colors, scheme]);

  let content: React.ReactNode;
  if (status === "booting") {
    content = <LoadingState label="Lueur" />;
  } else if (status === "signedOut") {
    content = <AuthScreens />;
  } else if (profile === undefined && profileError) {
    content = (
      <Screen footer={<Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />}>
        <ErrorState message={errorMessage(profileError)} onRetry={refetchProfile} />
      </Screen>
    );
  } else if (profile === undefined) {
    content = <LoadingState />;
  } else if (profile === null || !profile.isComplete || onboarding) {
    content = <OnboardingScreen onStart={startOnboarding} onFinish={finishOnboarding} />;
  } else {
    content = <AppScreens />;
  }

  return (
    <NavigationContainer ref={navigationRef} theme={navTheme}>
      {content}
    </NavigationContainer>
  );
}
