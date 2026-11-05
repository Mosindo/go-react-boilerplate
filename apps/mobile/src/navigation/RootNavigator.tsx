import React, { useRef, useState } from "react";
import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DefaultTheme, DarkTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthContext";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../lib/queryKeys";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import {
  ForgotScreen,
  LoginScreen,
  RegisterScreen,
  ResetScreen,
  WelcomeScreen
} from "../screens/AuthScreens";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { EditProfileScreen, PreferencesScreen } from "../screens/EditScreens";
import { MatchModal } from "../screens/MatchModal";
import { MeScreen } from "../screens/MeScreen";
import { MessagesScreen, useConversations } from "../screens/MessagesScreen";
import { NotificationsScreen, useUnreadCount } from "../screens/NotificationsScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { BlockedScreen, SettingsScreen } from "../screens/SettingsScreen";
import { UserProfileScreen } from "../screens/UserProfileScreen";
import { useTheme } from "../theme";
import { ErrorState, LoadingState } from "../ui";
import type { AuthStackParams, MainStackParams, TabParams } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tabs = createBottomTabNavigator<TabParams>();

function useNavTheme() {
  const t = useTheme();
  const base = t.mode === "dark" ? DarkTheme : DefaultTheme;
  return {
    ...base,
    colors: {
      ...base.colors,
      background: t.bg,
      card: t.surface,
      text: t.text,
      border: t.border,
      primary: t.primary
    }
  };
}

function TabsNavigator() {
  const t = useTheme();
  const conversations = useConversations();
  const unread = useUnreadCount();
  const unreadMessages = (conversations.data ?? []).reduce((n, c) => n + c.unreadCount, 0);
  const icon = (name: string, focused: boolean) =>
    (focused ? name : `${name}-outline`) as React.ComponentProps<typeof Ionicons>["name"];
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: t.primary,
        tabBarInactiveTintColor: t.textMuted,
        tabBarStyle: { backgroundColor: t.surface, borderTopColor: t.border }
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons name={icon("flame", focused)} size={size} color={color} />
          )
        }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons name={icon("chatbubbles", focused)} size={size} color={color} />
          )
        }}
      />
      <Tabs.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{
          title: "Activité",
          tabBarBadge: (unread.data ?? 0) > 0 ? unread.data : undefined,
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons name={icon("notifications", focused)} size={size} color={color} />
          )
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{
          title: "Profil",
          tabBarIcon: ({ color, size, focused }) => (
            <Ionicons name={icon("person", focused)} size={size} color={color} />
          )
        }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const t = useTheme();
  return (
    <RealtimeProvider>
      <MainStack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: t.surface },
          headerTintColor: t.text,
          headerShadowVisible: false,
          headerBackTitleVisible: false
        }}
      >
        <MainStack.Screen name="Tabs" component={TabsNavigator} options={{ headerShown: false }} />
        <MainStack.Screen name="Chat" component={ChatScreen} options={{ title: "" }} />
        <MainStack.Screen
          name="UserProfile"
          component={UserProfileScreen}
          options={{ title: "Profil" }}
        />
        <MainStack.Screen
          name="EditProfile"
          component={EditProfileScreen}
          options={{ title: "Modifier mon profil" }}
        />
        <MainStack.Screen
          name="Preferences"
          component={PreferencesScreen}
          options={{ title: "Préférences" }}
        />
        <MainStack.Screen
          name="Settings"
          component={SettingsScreen}
          options={{ title: "Paramètres" }}
        />
        <MainStack.Screen
          name="Blocked"
          component={BlockedScreen}
          options={{ title: "Personnes bloquées" }}
        />
      </MainStack.Navigator>
      <MatchModal />
    </RealtimeProvider>
  );
}

function AuthNavigator() {
  const t = useTheme();
  return (
    <AuthStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: t.bg },
        headerTintColor: t.text,
        headerShadowVisible: false,
        headerBackTitleVisible: false,
        title: ""
      }}
    >
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="Forgot" component={ForgotScreen} />
      <AuthStack.Screen name="Reset" component={ResetScreen} />
    </AuthStack.Navigator>
  );
}

/** Authenticated gate: onboarding until the server says the profile is complete. */
function SignedIn() {
  const overview = useQuery({ queryKey: queryKeys.overview, queryFn: profileApi.overview });
  const sawIncomplete = useRef(false);
  const [finished, setFinished] = useState(false);

  if (overview.isLoading) return <LoadingState />;
  if (overview.isError || !overview.data) {
    return (
      <ErrorState message={errorMessage(overview.error)} onRetry={() => void overview.refetch()} />
    );
  }
  if (!overview.data.onboardingComplete) sawIncomplete.current = true;
  // Keep the onboarding visible after the first photo so people can add more
  // until they explicitly finish.
  if (!overview.data.onboardingComplete || (sawIncomplete.current && !finished)) {
    return <OnboardingScreen overview={overview.data} onFinish={() => setFinished(true)} />;
  }
  return <MainNavigator />;
}

export function RootNavigator() {
  const { status, retryBoot } = useAuth();
  const navTheme = useNavTheme();
  if (status === "booting") return <LoadingState label="Ouverture de votre session…" />;
  if (status === "unreachable") {
    return (
      <ErrorState
        message="Impossible de joindre le serveur. Vérifiez votre connexion."
        onRetry={retryBoot}
      />
    );
  }
  return (
    <NavigationContainer theme={navTheme} key={status}>
      {status === "authenticated" ? <SignedIn /> : <AuthNavigator />}
    </NavigationContainer>
  );
}
