import React, { useMemo } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useConversations, useMyProfile, useUnreadNotifications } from "../hooks/queries";
import { RealtimeProvider, useRealtime } from "../hooks/useRealtime";
import { ForgotPasswordScreen } from "../screens/auth/ForgotPasswordScreen";
import { LoginScreen } from "../screens/auth/LoginScreen";
import { RegisterScreen } from "../screens/auth/RegisterScreen";
import { WelcomeScreen } from "../screens/auth/WelcomeScreen";
import { ChatScreen } from "../screens/chat/ChatScreen";
import { ConversationsScreen } from "../screens/chat/ConversationsScreen";
import { DiscoverScreen } from "../screens/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../screens/discover/ProfileDetailScreen";
import { ReportScreen } from "../screens/discover/ReportScreen";
import { NotificationsScreen } from "../screens/notifications/NotificationsScreen";
import { OnboardingScreen } from "../screens/onboarding/OnboardingScreen";
import { AccountScreen } from "../screens/profile/AccountScreen";
import {
  EditInterestsScreen,
  EditPhotosScreen,
  EditPreferencesScreen,
  EditProfileScreen
} from "../screens/profile/EditScreens";
import { MeScreen } from "../screens/profile/MeScreen";
import { PrivacyScreen } from "../screens/profile/PrivacyScreen";
import { useTheme } from "../theme/ThemeProvider";
import { Button, ErrorState, LoadingState, Screen } from "../ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function useNavTheme(): NavTheme {
  const { colors, scheme } = useTheme();
  return useMemo(() => {
    const base = scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: colors.primary,
        background: colors.background,
        card: colors.surface,
        text: colors.text,
        border: colors.border,
        notification: colors.primary
      }
    };
  }, [colors, scheme]);
}

function MainTabs() {
  const { colors } = useTheme();
  const { connected } = useRealtime();
  const conversations = useConversations(!connected);
  const unreadMessages =
    conversations.data?.pages.flatMap((p) => p.conversations).reduce((n, c) => n + (c.unreadCount > 0 ? 1 : 0), 0) ?? 0;
  const unreadNotifications = useUnreadNotifications();

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, minHeight: 60 },
        tabBarItemStyle: { paddingVertical: 4 },
        tabBarLabelStyle: { fontSize: 11, lineHeight: 14, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ color, focused, size }) => {
          const icons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
            Discover: ["flame", "flame-outline"],
            Matches: ["chatbubbles", "chatbubbles-outline"],
            Notifications: ["notifications", "notifications-outline"],
            Me: ["person-circle", "person-circle-outline"]
          };
          const [on, off] = icons[route.name];
          return <Ionicons name={focused ? on : off} size={size} color={color} />;
        }
      })}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{ title: "Découvrir", tabBarButtonTestID: "tab-discover" }}
      />
      <Tabs.Screen
        name="Matches"
        component={ConversationsScreen}
        options={{ title: "Matchs", tabBarBadge: unreadMessages || undefined, tabBarButtonTestID: "tab-matches" }}
      />
      <Tabs.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{
          title: "Activité",
          tabBarBadge: unreadNotifications || undefined,
          tabBarButtonTestID: "tab-notifications"
        }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarButtonTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function SignedInApp() {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const profile = useMyProfile();

  if (profile.isLoading) {
    return <LoadingState label="Chargement de ton profil…" />;
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen scroll={false}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </Screen>
    );
  }
  if (!profile.data.isComplete) {
    return <OnboardingScreen profile={profile.data} />;
  }

  const header = {
    headerShown: true,
    headerTintColor: colors.text,
    headerStyle: { backgroundColor: colors.surface },
    headerShadowVisible: false,
    headerBackButtonDisplayMode: "minimal" as const
  };

  return (
    <RealtimeProvider>
      <AppStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
        <AppStack.Screen name="Tabs" component={MainTabs} />
        <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ ...header, title: "" }} />
        <AppStack.Screen name="Chat" component={ChatScreen} options={header} />
        <AppStack.Screen
          name="Report"
          component={ReportScreen}
          options={{ ...header, title: "Signalement", presentation: "modal" }}
        />
        <AppStack.Screen
          name="EditProfile"
          component={EditProfileScreen}
          options={{ ...header, title: "Mes informations" }}
        />
        <AppStack.Screen name="EditPhotos" component={EditPhotosScreen} options={{ ...header, title: "Mes photos" }} />
        <AppStack.Screen
          name="EditPreferences"
          component={EditPreferencesScreen}
          options={{ ...header, title: "Préférences" }}
        />
        <AppStack.Screen
          name="EditInterests"
          component={EditInterestsScreen}
          options={{ ...header, title: "Centres d'intérêt" }}
        />
        <AppStack.Screen name="Privacy" component={PrivacyScreen} options={{ ...header, title: "Confidentialité" }} />
        <AppStack.Screen name="Account" component={AccountScreen} options={{ ...header, title: "Compte" }} />
      </AppStack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status } = useAuth();
  const { colors } = useTheme();
  const navTheme = useNavTheme();

  if (status === "booting") {
    return <LoadingState />;
  }
  return (
    <NavigationContainer theme={navTheme} documentTitle={{ formatter: () => "Seren" }}>
      {status === "signedOut" ? (
        <AuthStack.Navigator
          screenOptions={{
            headerTintColor: colors.text,
            headerStyle: { backgroundColor: colors.background },
            headerShadowVisible: false,
            headerTitle: "",
            headerBackButtonDisplayMode: "minimal",
            contentStyle: { backgroundColor: colors.background }
          }}
        >
          <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
          <AuthStack.Screen name="Login" component={LoginScreen} />
          <AuthStack.Screen name="Register" component={RegisterScreen} />
          <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
        </AuthStack.Navigator>
      ) : (
        <SignedInApp />
      )}
    </NavigationContainer>
  );
}
