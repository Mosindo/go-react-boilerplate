import React from "react";
import { Text as RNText } from "react-native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import {
  createNavigationContainerRef,
  DefaultTheme,
  NavigationContainer,
  type Theme
} from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthProvider";
import { MatchModal } from "../components/MatchModal";
import { isOnboarded, useConversations, useOwnProfile } from "../hooks/queries";
import { RealtimeProvider, useRealtime } from "../realtime/RealtimeProvider";
import OnboardingScreen from "../screens/OnboardingScreen";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import WelcomeScreen from "../screens/auth/WelcomeScreen";
import ChatScreen from "../screens/main/ChatScreen";
import ConversationsScreen from "../screens/main/ConversationsScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import { EditProfileScreen, PhotosScreen, PreferencesScreen } from "../screens/main/EditScreens";
import LikesScreen from "../screens/main/LikesScreen";
import MeScreen from "../screens/main/MeScreen";
import NotificationsScreen from "../screens/main/NotificationsScreen";
import ProfileDetailScreen from "../screens/main/ProfileDetailScreen";
import { BlockedUsersScreen, ReportScreen, SettingsScreen } from "../screens/main/SafetyScreens";
import { ErrorState, Loading } from "../ui";
import { Button } from "../ui/Button";
import { useTheme } from "../theme/ThemeProvider";
import type { AuthStackParams, MainStackParams, TabParams } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tabs = createBottomTabNavigator<TabParams>();

export const navigationRef = createNavigationContainerRef<MainStackParams>();

function tabIcon(symbol: string) {
  return function TabIcon({ color }: { color: string }) {
    return (
      <RNText style={{ color, fontSize: 20 }} accessibilityElementsHidden importantForAccessibility="no">
        {symbol}
      </RNText>
    );
  };
}

function TabNavigator() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const unreadMessages = (conversations.data?.pages ?? [])
    .flatMap((p) => p.conversations)
    .reduce((sum, c) => sum + c.unreadCount, 0);

  return (
    <Tabs.Navigator
      screenOptions={{
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        headerStyle: { backgroundColor: colors.background },
        headerTitleStyle: { color: colors.text },
        headerShadowVisible: false
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          headerShown: false,
          tabBarIcon: tabIcon("✦"),
          tabBarTestID: "tab-discover"
        }}
      />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{ title: "Likes", tabBarIcon: tabIcon("♥"), tabBarTestID: "tab-likes" }}
      />
      <Tabs.Screen
        name="Messages"
        component={ConversationsScreen}
        options={{
          title: "Messages",
          tabBarIcon: tabIcon("✉"),
          tabBarBadge: unreadMessages > 0 ? (unreadMessages > 99 ? "99+" : unreadMessages) : undefined,
          tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.accentText },
          tabBarTestID: "tab-messages"
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{ title: "Profil", tabBarIcon: tabIcon("☺"), tabBarTestID: "tab-me" }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const { colors } = useTheme();
  return (
    <MainStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTitleStyle: { color: colors.text },
        headerTintColor: colors.accent,
        headerShadowVisible: false,
        headerBackTitle: "Retour",
        contentStyle: { backgroundColor: colors.background }
      }}
    >
      <MainStack.Screen name="Tabs" component={TabNavigator} options={{ headerShown: false }} />
      <MainStack.Screen name="Chat" component={ChatScreen} />
      <MainStack.Screen name="ProfileDetail" component={ProfileDetailScreen} />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: "Modifier mon profil" }}
      />
      <MainStack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
      <MainStack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Préférences" }} />
      <MainStack.Screen name="Notifications" component={NotificationsScreen} />
      <MainStack.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: "Confidentialité et compte" }}
      />
      <MainStack.Screen
        name="BlockedUsers"
        component={BlockedUsersScreen}
        options={{ title: "Utilisateurs bloqués" }}
      />
      <MainStack.Screen
        name="Report"
        component={ReportScreen}
        options={{ title: "Signaler", presentation: "modal" }}
      />
    </MainStack.Navigator>
  );
}

/** Shows the "it's a match" sheet when the other person completes a match while the app is open. */
function IncomingMatch() {
  const { incomingMatch, dismissMatch } = useRealtime();
  return (
    <MatchModal
      match={incomingMatch}
      onClose={dismissMatch}
      onMessage={(m) => {
        dismissMatch();
        if (navigationRef.isReady())
          navigationRef.navigate("Chat", { conversationId: m.conversationId, name: m.profile.firstName });
      }}
    />
  );
}

function SignedIn() {
  const profile = useOwnProfile();
  const { signOut } = useAuth();

  if (profile.isPending) return <Loading label="Chargement de votre profil…" />;
  if (profile.isError) {
    return (
      <>
        <ErrorState error={profile.error} onRetry={() => void profile.refetch()} />
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </>
    );
  }
  if (!isOnboarded(profile.data)) return <OnboardingScreen profile={profile.data ?? null} />;

  return (
    <RealtimeProvider>
      <MainNavigator />
      <IncomingMatch />
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status } = useAuth();
  const { colors, isDark } = useTheme();
  const theme: Theme = {
    ...DefaultTheme,
    dark: isDark,
    colors: {
      ...DefaultTheme.colors,
      primary: colors.accent,
      background: colors.background,
      card: colors.surface,
      text: colors.text,
      border: colors.border,
      notification: colors.accent
    }
  };

  return (
    <NavigationContainer ref={navigationRef} theme={theme}>
      {status === "booting" ? (
        <Loading label="Ouverture de votre session…" />
      ) : status === "signedOut" ? (
        <AuthStack.Navigator
          screenOptions={{
            headerStyle: { backgroundColor: colors.background },
            headerTitleStyle: { color: colors.text },
            headerTintColor: colors.accent,
            headerShadowVisible: false,
            headerBackTitle: "Retour",
            contentStyle: { backgroundColor: colors.background }
          }}
        >
          <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
          <AuthStack.Screen name="Login" component={LoginScreen} options={{ title: "Connexion" }} />
          <AuthStack.Screen name="Register" component={RegisterScreen} options={{ title: "Inscription" }} />
          <AuthStack.Screen
            name="ForgotPassword"
            component={ForgotPasswordScreen}
            options={{ title: "Mot de passe oublié" }}
          />
        </AuthStack.Navigator>
      ) : (
        <SignedIn />
      )}
    </NavigationContainer>
  );
}
