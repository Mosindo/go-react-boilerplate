import React, { useMemo } from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useAuth } from "../providers/AuthProvider";
import { RealtimeProvider } from "../providers/RealtimeProvider";
import { useTheme } from "../theme";
import { LoadingState, Screen } from "../ui";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { ResetPasswordScreen } from "../features/auth/ResetPasswordScreen";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { ChatScreen } from "../features/chat/ChatScreen";
import { ConversationsScreen, useConversations } from "../features/chat/ConversationsScreen";
import { DiscoverScreen } from "../features/discovery/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discovery/ProfileDetailScreen";
import { LikesScreen } from "../features/likes/LikesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { ChangePasswordScreen } from "../features/profile/ChangePasswordScreen";
import { DeleteAccountScreen } from "../features/profile/DeleteAccountScreen";
import { EditPhotosScreen } from "../features/profile/EditPhotosScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { SettingsScreen } from "../features/profile/SettingsScreen";
import { BlockedUsersScreen } from "../features/safety/BlockedUsersScreen";
import { ReportScreen } from "../features/safety/ReportScreen";
import type { AppStackParamList, AuthStackParamList, MainTabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

const tabIcons: Record<keyof MainTabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
  Discover: ["flame", "flame-outline"],
  Likes: ["heart", "heart-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const active = useConversations("active");
  const unread = (active.data ?? []).filter((c) => c.unreadCount > 0).length;
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarIcon: ({ focused, color, size }) => <Ionicons color={color} name={tabIcons[route.name][focused ? 0 : 1]} size={size} />
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen component={LikesScreen} name="Likes" options={{ title: "Likes", tabBarTestID: "tab-likes" }} />
      <Tabs.Screen
        component={ConversationsScreen}
        name="Matches"
        options={{
          title: "Matchs",
          tabBarTestID: "tab-matches",
          tabBarBadge: unread || undefined,
          tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary }
        }}
      />
      <Tabs.Screen component={MeScreen} name="Me" options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function SignedInApp() {
  return (
    <AppStack.Navigator screenOptions={{ headerShown: false, animation: "slide_from_right" }}>
      <AppStack.Screen component={MainTabs} name="Tabs" />
      <AppStack.Screen component={ProfileDetailScreen} name="ProfileDetail" />
      <AppStack.Screen component={ChatScreen} name="Chat" />
      <AppStack.Screen component={EditProfileScreen} name="EditProfile" />
      <AppStack.Screen component={EditPhotosScreen} name="EditPhotos" />
      <AppStack.Screen component={PreferencesScreen} name="Preferences" />
      <AppStack.Screen component={SettingsScreen} name="Settings" />
      <AppStack.Screen component={BlockedUsersScreen} name="BlockedUsers" />
      <AppStack.Screen component={ChangePasswordScreen} name="ChangePassword" />
      <AppStack.Screen component={DeleteAccountScreen} name="DeleteAccount" />
      <AppStack.Screen component={NotificationsScreen} name="Notifications" />
      <AppStack.Screen component={ReportScreen} name="Report" options={{ presentation: "modal" }} />
    </AppStack.Navigator>
  );
}

function SignedOutApp() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false, animation: "slide_from_right" }}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" />
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
      <AuthStack.Screen component={ResetPasswordScreen} name="ResetPassword" />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, user } = useAuth();
  const theme = useTheme();
  const navTheme = useMemo<NavTheme>(() => {
    const base = theme.scheme === "dark" ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        background: theme.colors.background,
        card: theme.colors.surface,
        border: theme.colors.border,
        primary: theme.colors.primary,
        text: theme.colors.text
      }
    };
  }, [theme]);

  if (status === "booting") {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  const signedIn = status === "signedIn" && !!user;
  const complete = signedIn && user.profileComplete;
  return (
    <RealtimeProvider enabled={complete}>
      <NavigationContainer theme={navTheme}>
        {!signedIn ? <SignedOutApp /> : !complete ? <OnboardingScreen /> : <SignedInApp key={user.id} />}
      </NavigationContainer>
    </RealtimeProvider>
  );
}
