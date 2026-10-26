import React from "react";
import {
  DefaultTheme,
  DarkTheme,
  NavigationContainer,
  createNavigationContainerRef
} from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { useAuth } from "../auth/AuthContext";
import { MatchModal } from "../components/MatchModal";
import { ErrorState, LoadingState } from "../components/States";
import { Screen } from "../components/Screen";
import { useConversations, useOwnProfile, useUnreadCount } from "../hooks/useQueries";
import { useRealtime } from "../realtime/RealtimeProvider";
import { ActivityScreen } from "../screens/ActivityScreen";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { ForgotScreen } from "../screens/ForgotScreen";
import { MeScreen } from "../screens/MeScreen";
import { MessagesScreen } from "../screens/MessagesScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { ProfileDetailScreen } from "../screens/ProfileDetailScreen";
import {
  AccountScreen,
  BlockedScreen,
  EditProfileScreen,
  PhotosScreen,
  PreferencesScreen,
  PrivacyScreen
} from "../screens/SettingsScreens";
import { WelcomeScreen } from "../screens/WelcomeScreen";
import { useTheme } from "../theme";
import type { AuthStackParamList, RootStackParamList, TabParamList } from "./types";

const navigationRef = createNavigationContainerRef<RootStackParamList>();
const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const RootStack = createNativeStackNavigator<RootStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function MainTabs() {
  const theme = useTheme();
  const unread = useUnreadCount(true);
  const conversations = useConversations();
  const unreadMessages = (conversations.data?.pages.flatMap((p) => p.conversations) ?? []).reduce(
    (n, c) => n + c.unreadCount,
    0
  );

  const icon = (name: keyof typeof Ionicons.glyphMap) =>
    function TabIcon({ color, size }: { color: string; size: number }) {
      return <Ionicons name={name} color={color} size={size} />;
    };

  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: theme.primary,
        tabBarInactiveTintColor: theme.textMuted,
        tabBarStyle: { backgroundColor: theme.surface, borderTopColor: theme.border },
        tabBarBadgeStyle: { backgroundColor: theme.primary, color: theme.onPrimary }
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          tabBarIcon: icon("heart-outline"),
          tabBarButtonTestID: "tab-discover"
        }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarIcon: icon("chatbubbles-outline"),
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarButtonTestID: "tab-messages"
        }}
      />
      <Tabs.Screen
        name="Activity"
        component={ActivityScreen}
        options={{
          title: "Activité",
          tabBarIcon: icon("notifications-outline"),
          tabBarBadge: (unread.data ?? 0) > 0 ? unread.data : undefined,
          tabBarButtonTestID: "tab-activity"
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{ title: "Profil", tabBarIcon: icon("person-outline"), tabBarButtonTestID: "tab-me" }}
      />
    </Tabs.Navigator>
  );
}

function SignedInStack() {
  const theme = useTheme();
  const { incomingMatch, dismissMatch } = useRealtime();
  const header = {
    headerStyle: { backgroundColor: theme.bg },
    headerTintColor: theme.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour"
  };

  return (
    <>
      <RootStack.Navigator screenOptions={header}>
        <RootStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
        <RootStack.Screen name="Chat" component={ChatScreen} options={{ title: "" }} />
        <RootStack.Screen
          name="ProfileDetail"
          component={ProfileDetailScreen}
          options={{ title: "Profil" }}
        />
        <RootStack.Screen
          name="EditProfile"
          component={EditProfileScreen}
          options={{ title: "Modifier mon profil" }}
        />
        <RootStack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
        <RootStack.Screen
          name="Preferences"
          component={PreferencesScreen}
          options={{ title: "Mes préférences" }}
        />
        <RootStack.Screen name="Privacy" component={PrivacyScreen} options={{ title: "Confidentialité" }} />
        <RootStack.Screen
          name="Blocked"
          component={BlockedScreen}
          options={{ title: "Personnes bloquées" }}
        />
        <RootStack.Screen
          name="Account"
          component={AccountScreen}
          options={{ title: "Compte et sécurité" }}
        />
      </RootStack.Navigator>
      <MatchModal
        match={incomingMatch}
        onClose={dismissMatch}
        onChat={() => {
          if (incomingMatch) {
            navigationRef.navigate("Chat", { matchId: incomingMatch.id, profile: incomingMatch.profile });
          }
          dismissMatch();
        }}
      />
    </>
  );
}

/** Signed-in users see onboarding until their profile is complete, then the app. */
function SignedInGate() {
  const profile = useOwnProfile();
  if (profile.isPending)
    return (
      <Screen>
        <LoadingState label="Chargement de votre profil…" />
      </Screen>
    );
  if (profile.isError)
    return (
      <Screen>
        <ErrorState message={profile.error.message} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  if (!profile.data?.isComplete) return <OnboardingScreen profile={profile.data} />;
  return <SignedInStack />;
}

export function AppNavigator() {
  const theme = useTheme();
  const { status, retryBoot } = useAuth();
  const base = theme.scheme === "dark" ? DarkTheme : DefaultTheme;
  const navTheme = {
    ...base,
    colors: {
      ...base.colors,
      background: theme.bg,
      card: theme.bg,
      text: theme.text,
      border: theme.border,
      primary: theme.primary
    }
  };

  let content: React.ReactNode;
  if (status === "booting")
    content = (
      <Screen>
        <LoadingState />
      </Screen>
    );
  else if (status === "failed")
    content = (
      <Screen>
        <ErrorState message="Impossible de joindre le serveur." onRetry={retryBoot} />
      </Screen>
    );
  else if (status === "signedOut")
    content = (
      <AuthStack.Navigator
        screenOptions={{
          headerShadowVisible: false,
          headerStyle: { backgroundColor: theme.bg },
          headerTintColor: theme.text
        }}
      >
        <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
        <AuthStack.Screen
          name="Forgot"
          component={ForgotScreen}
          options={{ title: "", headerBackTitle: "Retour" }}
        />
      </AuthStack.Navigator>
    );
  else content = <SignedInGate />;

  return (
    <NavigationContainer ref={navigationRef} theme={navTheme}>
      {content}
    </NavigationContainer>
  );
}
