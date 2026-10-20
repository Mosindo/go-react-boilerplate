import React from "react";
import { DefaultTheme, DarkTheme, NavigationContainer } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme";
import { LoadingState } from "../ui";
import { useAuth } from "../features/auth/AuthProvider";
import {
  ForgotPasswordScreen,
  LoginScreen,
  RegisterScreen,
  ResetPasswordScreen
} from "../features/auth/AuthScreens";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discover/ProfileDetailScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { ChatsScreen, useConversations } from "../features/chat/ChatsScreen";
import { ThreadScreen } from "../features/chat/ThreadScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { MeScreen } from "../features/profile/MeScreen";
import {
  BlockedScreen,
  EditProfileScreen,
  PhotosScreen,
  PreferencesScreen
} from "../features/profile/ProfileSettingsScreens";
import { ReportScreen } from "../features/safety/ReportScreen";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import type { AuthStackParamList, MainStackParamList, MainTabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

const ICONS: Record<keyof MainTabParamList, keyof typeof Ionicons.glyphMap> = {
  Discover: "flame-outline",
  Matches: "heart-outline",
  Chats: "chatbubbles-outline",
  Me: "person-outline"
};

function TabsNavigator() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const unread =
    conversations.data?.pages.flatMap((p) => p.conversations).reduce((sum, c) => sum + c.unreadCount, 0) ?? 0;
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarIcon: ({ color, size }) => <Ionicons name={ICONS[route.name]} size={size} color={color} />
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} />
      <Tabs.Screen name="Matches" component={MatchesScreen} />
      <Tabs.Screen
        name="Chats"
        component={ChatsScreen}
        options={{
          tabBarBadge: unread > 0 ? (unread > 9 ? "9+" : unread) : undefined,
          tabBarBadgeStyle: { backgroundColor: colors.accent }
        }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profile" }} />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  return (
    <MainStack.Navigator>
      <MainStack.Screen name="Tabs" component={TabsNavigator} options={{ headerShown: false }} />
      <MainStack.Screen name="Thread" component={ThreadScreen} />
      <MainStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "Profile" }} />
      <MainStack.Screen
        name="Report"
        component={ReportScreen}
        options={{ presentation: "modal", title: "Report" }}
      />
      <MainStack.Screen name="Notifications" component={NotificationsScreen} />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: "Edit profile" }}
      />
      <MainStack.Screen name="Photos" component={PhotosScreen} />
      <MainStack.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{ title: "Discovery & privacy" }}
      />
      <MainStack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Blocked people" }} />
    </MainStack.Navigator>
  );
}

function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { colors, isDark } = useTheme();
  const { status, user } = useAuth();
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = {
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

  let content: React.ReactNode;
  if (status === "booting") content = <LoadingState label="Restoring your session" />;
  else if (status === "anonymous" || !user) content = <AuthNavigator />;
  else if (!user.profileComplete) content = <OnboardingScreen />;
  else
    content = (
      <RealtimeProvider userId={user.id}>
        <MainNavigator />
      </RealtimeProvider>
    );

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
