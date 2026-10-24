import React from "react";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { chatApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthProvider";
import {
  ForgotPasswordScreen,
  LoginScreen,
  RegisterScreen,
  WelcomeScreen,
} from "../features/auth/AuthScreens";
import { MessagesScreen } from "../features/chat/MessagesScreen";
import { ChatScreen } from "../features/chat/ChatScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { MatchFoundScreen } from "../features/discover/MatchFoundScreen";
import { ProfileDetailScreen } from "../features/discover/ProfileDetailScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { ProfileScreen } from "../features/profile/ProfileScreen";
import {
  BlockedUsersScreen,
  DeleteAccountScreen,
  EditProfileScreen,
  ManagePhotosScreen,
  PreferencesScreen,
  PrivacyScreen,
} from "../features/profile/SettingsScreens";
import { queryKeys } from "../realtime/cache";
import { useTheme } from "../theme/theme";
import { Icon, LoadingState } from "../ui";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function TabNavigator() {
  const { colors } = useTheme();
  const { data: unread = 0 } = useQuery({
    queryKey: queryKeys.chatUnread,
    queryFn: chatApi.unread,
  });
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          tabBarTestID: "tab-discover",
          tabBarIcon: ({ color, size }) => <Icon name="flame-outline" size={size} color={color} />,
        }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarTestID: "tab-messages",
          tabBarBadge: unread > 0 ? (unread > 99 ? "99+" : unread) : undefined,
          tabBarIcon: ({ color, size }) => (
            <Icon name="chatbubble-ellipses-outline" size={size} color={color} />
          ),
        }}
      />
      <Tabs.Screen
        name="Profile"
        component={ProfileScreen}
        options={{
          title: "Profil",
          tabBarTestID: "tab-profile",
          tabBarIcon: ({ color, size }) => (
            <Icon name="person-circle-outline" size={size} color={color} />
          ),
        }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const { colors } = useTheme();
  const header = {
    headerStyle: { backgroundColor: colors.bg },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour",
  } as const;
  return (
    <MainStack.Navigator screenOptions={header}>
      <MainStack.Screen name="Tabs" component={TabNavigator} options={{ headerShown: false }} />
      <MainStack.Screen
        name="ProfileDetail"
        component={ProfileDetailScreen}
        options={{ headerShown: false }}
      />
      <MainStack.Screen name="Chat" component={ChatScreen} options={{ title: "" }} />
      <MainStack.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{ title: "Notifications" }}
      />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: "Mon profil" }}
      />
      <MainStack.Screen
        name="ManagePhotos"
        component={ManagePhotosScreen}
        options={{ title: "Mes photos" }}
      />
      <MainStack.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{ title: "Préférences" }}
      />
      <MainStack.Screen
        name="Privacy"
        component={PrivacyScreen}
        options={{ title: "Confidentialité" }}
      />
      <MainStack.Screen
        name="BlockedUsers"
        component={BlockedUsersScreen}
        options={{ title: "Personnes bloquées" }}
      />
      <MainStack.Screen
        name="DeleteAccount"
        component={DeleteAccountScreen}
        options={{ title: "Supprimer mon compte" }}
      />
      <MainStack.Screen
        name="MatchFound"
        component={MatchFoundScreen}
        options={{ presentation: "transparentModal", headerShown: false, animation: "fade" }}
      />
    </MainStack.Navigator>
  );
}

function AuthNavigator() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.bg },
        headerTintColor: colors.text,
        headerShadowVisible: false,
        headerBackTitle: "Retour",
        title: "",
      }}
    >
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { colors, scheme } = useTheme();
  const { status, user } = useAuth();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  const theme: Theme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.bg,
      card: colors.bg,
      text: colors.text,
      border: colors.border,
      primary: colors.primary,
    },
  };

  let content: React.ReactNode;
  if (status === "booting") content = <LoadingState label="Chargement…" />;
  else if (status === "signedOut" || !user) content = <AuthNavigator />;
  else if (!user.profileComplete) content = <OnboardingScreen />;
  else content = <MainNavigator />;

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
