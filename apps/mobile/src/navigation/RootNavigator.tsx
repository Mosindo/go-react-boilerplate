import React, { useEffect, useState } from "react";
import { Text } from "react-native";
import { NavigationContainer, DefaultTheme, DarkTheme, useNavigation } from "@react-navigation/native";
import { createNativeStackNavigator, type NativeStackNavigationProp } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../auth/AuthContext";
import { useRealtime } from "../realtime/RealtimeProvider";
import { useMyProfile } from "../hooks";
import { chatApi, discoveryApi } from "../api/services";
import { ErrorState, Loading, Screen } from "../components/ui";
import { MatchModal } from "../components/Modals";
import { useTheme } from "../theme/theme";
import AuthScreen from "../screens/auth/AuthScreen";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import LikesScreen from "../screens/main/LikesScreen";
import MessagesScreen from "../screens/main/MessagesScreen";
import MeScreen from "../screens/main/MeScreen";
import ProfileDetailScreen from "../screens/main/ProfileDetailScreen";
import ChatScreen from "../screens/main/ChatScreen";
import EditProfileScreen from "../screens/main/EditProfileScreen";
import SettingsScreen from "../screens/main/SettingsScreen";
import BlockedScreen from "../screens/main/BlockedScreen";
import NotificationsScreen from "../screens/main/NotificationsScreen";
import type { AuthStackParams, MainStackParams, TabParams } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tabs = createBottomTabNavigator<TabParams>();

function TabLabel({
  glyph,
  label,
  focused,
  color,
}: {
  glyph: string;
  label: string;
  focused: boolean;
  color: string;
}) {
  return (
    <Text accessibilityLabel={label} style={{ color, fontSize: 11, fontWeight: focused ? "800" : "600" }}>
      {glyph} {label}
    </Text>
  );
}

function MainTabs() {
  const { colors } = useTheme();
  const likes = useQuery({ queryKey: ["likes"], queryFn: discoveryApi.likesReceived });
  const convs = useQuery({ queryKey: ["conversations"], queryFn: chatApi.conversations });
  const unreadMessages = (convs.data ?? []).reduce((n, c) => n + c.unreadCount, 0);
  const likesCount = likes.data?.length ?? 0;

  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarShowLabel: false,
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          tabBarTestID: "tab-discover",
          tabBarIcon: ({ color, focused }) => (
            <TabLabel glyph="✦" label="Découvrir" focused={focused} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{
          tabBarTestID: "tab-likes",
          tabBarBadge: likesCount > 0 ? likesCount : undefined,
          tabBarIcon: ({ color, focused }) => (
            <TabLabel glyph="♥" label="Likes" focused={focused} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          tabBarTestID: "tab-messages",
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarIcon: ({ color, focused }) => (
            <TabLabel glyph="✉" label="Messages" focused={focused} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{
          tabBarTestID: "tab-me",
          tabBarIcon: ({ color, focused }) => (
            <TabLabel glyph="☺" label="Profil" focused={focused} color={color} />
          ),
        }}
      />
    </Tabs.Navigator>
  );
}

function RealtimeMatchModal() {
  const { incomingMatch, dismissMatch } = useRealtime();
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  return (
    <MatchModal
      match={incomingMatch}
      onClose={dismissMatch}
      onChat={() => {
        const m = incomingMatch;
        dismissMatch();
        if (m)
          nav.navigate("Chat", {
            conversationId: m.conversationId,
            peerName: m.user.firstName,
            peerId: m.user.userId,
          });
      }}
    />
  );
}

function MainNavigator() {
  const { colors } = useTheme();
  const opts = {
    headerStyle: { backgroundColor: colors.bg },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour",
  };
  return (
    <>
      <MainStack.Navigator screenOptions={opts}>
        <MainStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
        <MainStack.Screen
          name="ProfileDetail"
          component={ProfileDetailScreen}
          options={{ title: "Profil" }}
        />
        <MainStack.Screen name="Chat" component={ChatScreen} />
        <MainStack.Screen
          name="EditProfile"
          component={EditProfileScreen}
          options={{ title: "Mon profil" }}
        />
        <MainStack.Screen name="Settings" component={SettingsScreen} options={{ title: "Compte" }} />
        <MainStack.Screen
          name="Blocked"
          component={BlockedScreen}
          options={{ title: "Personnes bloquées" }}
        />
        <MainStack.Screen
          name="Notifications"
          component={NotificationsScreen}
          options={{ title: "Notifications" }}
        />
      </MainStack.Navigator>
      <RealtimeMatchModal />
    </>
  );
}

function Gate() {
  const { status } = useAuth();
  const profile = useMyProfile();
  // Keep onboarding on screen until the user taps "finish", even once the first photo makes the profile complete.
  const needsOnboarding =
    status === "signedIn" &&
    !profile.isLoading &&
    !profile.isError &&
    (!profile.data || profile.data.photos.length === 0);
  const [inOnboarding, setInOnboarding] = useState(false);
  useEffect(() => {
    if (needsOnboarding) setInOnboarding(true);
    if (status === "signedOut") setInOnboarding(false);
  }, [needsOnboarding, status]);

  if (status === "booting")
    return (
      <Screen>
        <Loading label="Chargement…" />
      </Screen>
    );
  if (status === "signedOut") {
    return (
      <AuthStack.Navigator
        screenOptions={{ headerShadowVisible: false, headerTitle: "", headerBackTitle: "Retour" }}
      >
        <AuthStack.Screen name="Auth" component={AuthScreen} options={{ headerShown: false }} />
        <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      </AuthStack.Navigator>
    );
  }
  if (profile.isLoading)
    return (
      <Screen>
        <Loading label="Chargement de votre profil…" />
      </Screen>
    );
  if (profile.isError)
    return (
      <Screen>
        <ErrorState
          message="Impossible de charger votre profil. Vérifiez votre connexion."
          onRetry={() => void profile.refetch()}
        />
      </Screen>
    );
  if (needsOnboarding || inOnboarding)
    return <OnboardingScreen existing={profile.data ?? null} onFinish={() => setInOnboarding(false)} />;
  return <MainNavigator />;
}

export default function RootNavigator() {
  const { isDark, colors } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  return (
    <NavigationContainer
      theme={{
        ...base,
        colors: {
          ...base.colors,
          background: colors.bg,
          card: colors.bg,
          text: colors.text,
          border: colors.border,
          primary: colors.primary,
        },
      }}
    >
      <Gate />
    </NavigationContainer>
  );
}
