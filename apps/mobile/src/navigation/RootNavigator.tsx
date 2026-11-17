import React, { useCallback, useRef } from "react";
import { DarkTheme, NavigationContainer, createNavigationContainerRef } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { useAuth } from "../auth/AuthProvider";
import { useMyProfile, useUnreadConversations, useUnreadNotifications } from "../hooks";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import { colors } from "../theme";
import { ErrorState, LoadingState, Screen } from "../ui";
import type { RootStackParamList, TabParamList } from "./types";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { ForgotPasswordScreen, ResetPasswordScreen } from "../features/auth/PasswordResetScreens";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discover/ProfileDetailScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { ConversationScreen } from "../features/chat/ConversationScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { PhotosScreen } from "../features/profile/PhotosScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { PrivacyScreen } from "../features/profile/PrivacyScreen";
import { ChangePasswordScreen, DeleteAccountScreen } from "../features/profile/AccountScreens";
import { BlockedScreen } from "../features/safety/BlockedScreen";
import { ReportScreen } from "../features/safety/ReportScreen";

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();
export const navigationRef = createNavigationContainerRef<RootStackParamList>();

const navigationTheme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    background: colors.bg,
    card: colors.bg,
    border: colors.border,
    primary: colors.accent,
    text: colors.text,
    notification: colors.accent
  }
};

const stackScreenOptions = {
  headerStyle: { backgroundColor: colors.bg },
  headerTintColor: colors.text,
  headerShadowVisible: false,
  headerBackButtonDisplayMode: "minimal" as const,
  headerBackTitle: "Retour",
  contentStyle: { backgroundColor: colors.bg }
};

type TabIcon = React.ComponentProps<typeof Ionicons>["name"];
const tabIcons: Record<keyof TabParamList, [TabIcon, TabIcon]> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Activity: ["notifications", "notifications-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const unreadChats = useUnreadConversations();
  const unreadNotifications = useUnreadNotifications();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarStyle: { backgroundColor: colors.bg, borderTopColor: colors.border, height: 64, paddingBottom: 8, paddingTop: 6 },
        tabBarActiveTintColor: colors.accent,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarLabelStyle: { fontSize: 11.5, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.accentText, fontSize: 11 },
        tabBarIcon: ({ focused, color, size }) => <Ionicons name={tabIcons[route.name][focused ? 0 : 1]} size={size} color={color} />
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir", tabBarButtonTestID: "tab-discover" }} />
      <Tabs.Screen
        name="Matches"
        component={MatchesScreen}
        options={{
          title: "Matchs",
          tabBarButtonTestID: "tab-matches",
          tabBarBadge: unreadChats.data ? unreadChats.data : undefined
        }}
      />
      <Tabs.Screen
        name="Activity"
        component={NotificationsScreen}
        options={{
          title: "Activité",
          tabBarButtonTestID: "tab-activity",
          tabBarBadge: unreadNotifications.data ? unreadNotifications.data : undefined
        }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarButtonTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function GuestStack() {
  return (
    <Stack.Navigator screenOptions={stackScreenOptions}>
      <Stack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <Stack.Screen name="Login" component={LoginScreen} options={{ title: "Connexion" }} />
      <Stack.Screen name="Register" component={RegisterScreen} options={{ title: "Créer un compte" }} />
      <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} options={{ title: "Mot de passe oublié" }} />
      <Stack.Screen name="ResetPassword" component={ResetPasswordScreen} options={{ title: "Nouveau mot de passe" }} />
    </Stack.Navigator>
  );
}

function MemberStack() {
  const profile = useMyProfile();
  const activeConversationRef = useRef<string | null>(null);
  const openConversation = useCallback((conversationId: string) => {
    if (navigationRef.isReady()) navigationRef.navigate("Conversation", { conversationId });
  }, []);

  if (profile.isLoading) {
    return (
      <Screen>
        <LoadingState label="Chargement de ton profil…" />
      </Screen>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen>
        <ErrorState onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }

  if (!profile.data.isComplete) {
    return (
      <Stack.Navigator screenOptions={stackScreenOptions}>
        <Stack.Screen name="Onboarding" component={OnboardingScreen} options={{ headerShown: false }} />
      </Stack.Navigator>
    );
  }

  return (
    <RealtimeProvider onOpenConversation={openConversation} activeConversationRef={activeConversationRef}>
      <ActiveConversationContext.Provider value={activeConversationRef}>
        <Stack.Navigator screenOptions={stackScreenOptions}>
          <Stack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
          <Stack.Screen name="Conversation" component={ConversationScreen} options={{ title: "" }} />
          <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "", presentation: "modal" }} />
          <Stack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier mon profil" }} />
          <Stack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
          <Stack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Préférences" }} />
          <Stack.Screen name="Privacy" component={PrivacyScreen} options={{ title: "Confidentialité" }} />
          <Stack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Membres bloqués" }} />
          <Stack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ title: "Mot de passe" }} />
          <Stack.Screen name="DeleteAccount" component={DeleteAccountScreen} options={{ title: "Supprimer mon compte" }} />
          <Stack.Screen name="Report" component={ReportScreen} options={{ title: "Signaler", presentation: "modal" }} />
        </Stack.Navigator>
      </ActiveConversationContext.Provider>
    </RealtimeProvider>
  );
}

export const ActiveConversationContext = React.createContext<{ current: string | null }>({ current: null });

export function RootNavigator() {
  const { state } = useAuth();
  return (
    <NavigationContainer ref={navigationRef} theme={navigationTheme} documentTitle={{ formatter: () => "Lueur" }}>
      {state.status === "booting" ? (
        <Screen>
          <LoadingState />
        </Screen>
      ) : state.status === "guest" ? (
        <GuestStack />
      ) : (
        <MemberStack key={state.user.id} />
      )}
    </NavigationContainer>
  );
}
