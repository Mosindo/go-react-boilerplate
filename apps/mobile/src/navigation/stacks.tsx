import React from "react";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme";
import { ForgotPasswordScreen, LoginScreen, RegisterScreen, WelcomeScreen } from "../features/auth/screens";
import {
  AboutScreen,
  BasicsScreen,
  LocationScreen,
  PhotosScreen,
  PreferencesScreen
} from "../features/onboarding/screens";
import { DiscoveryScreen } from "../features/discovery/DiscoveryScreen";
import { MatchesScreen } from "../features/matches/MatchesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { useUnreadNotificationCount } from "../features/notifications/hooks";
import { useUnreadMessageCount } from "../features/chat/hooks";
import { ProfileScreen } from "../features/profile/ProfileScreen";
import { ChatScreen } from "../features/chat/ChatScreen";
import { UserProfileScreen } from "../features/profile/UserProfileScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { EditPreferencesScreen } from "../features/profile/EditPreferencesScreen";
import { PrivacyScreen } from "../features/profile/PrivacyScreen";
import { BlockedUsersScreen } from "../features/profile/BlockedUsersScreen";
import { ChangePasswordScreen, DeleteAccountScreen } from "../features/profile/AccountScreens";
import type { AuthStackParamList, MainStackParamList, MainTabParamList, OnboardingStackParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const OnboardingStack = createNativeStackNavigator<OnboardingStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<MainTabParamList>();

export function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShadowVisible: false, headerTitle: "", headerBackTitle: "Retour" }}>
      <AuthStack.Screen component={WelcomeScreen} name="Welcome" options={{ headerShown: false }} />
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
    </AuthStack.Navigator>
  );
}

export function OnboardingNavigator() {
  return (
    <OnboardingStack.Navigator screenOptions={{ headerShown: false, gestureEnabled: false }}>
      <OnboardingStack.Screen component={BasicsScreen} name="Basics" />
      <OnboardingStack.Screen component={AboutScreen} name="About" />
      <OnboardingStack.Screen component={PreferencesScreen} name="Preferences" />
      <OnboardingStack.Screen component={PhotosScreen} name="Photos" />
      <OnboardingStack.Screen component={LocationScreen} name="Location" />
    </OnboardingStack.Navigator>
  );
}

const TAB_ICONS: Record<
  keyof MainTabParamList,
  [active: React.ComponentProps<typeof Ionicons>["name"], inactive: React.ComponentProps<typeof Ionicons>["name"]]
> = {
  Discover: ["flame", "flame-outline"],
  Matches: ["chatbubbles", "chatbubbles-outline"],
  Notifications: ["notifications", "notifications-outline"],
  Profile: ["person", "person-outline"]
};

function TabsNavigator() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessageCount().data ?? 0;
  const unreadNotifications = useUnreadNotificationCount();

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons color={color} name={TAB_ICONS[route.name][focused ? 0 : 1]} size={size} />
        )
      })}
    >
      <Tabs.Screen
        component={DiscoveryScreen}
        name="Discover"
        options={{ title: "Découvrir", tabBarTestID: "tab-discover" }}
      />
      <Tabs.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          title: "Matchs",
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarTestID: "tab-matches"
        }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          title: "Notifications",
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarTestID: "tab-notifications"
        }}
      />
      <Tabs.Screen
        component={ProfileScreen}
        name="Profile"
        options={{ title: "Profil", tabBarTestID: "tab-profile" }}
      />
    </Tabs.Navigator>
  );
}

export function MainNavigator() {
  const { colors } = useTheme();
  return (
    <MainStack.Navigator
      screenOptions={{
        headerShadowVisible: false,
        headerBackTitle: "Retour",
        headerTintColor: colors.primary,
        headerTitleStyle: { color: colors.text }
      }}
    >
      <MainStack.Screen component={TabsNavigator} name="Tabs" options={{ headerShown: false }} />
      <MainStack.Screen component={ChatScreen} name="Chat" options={{ headerShown: false }} />
      <MainStack.Screen component={UserProfileScreen} name="UserProfile" options={{ headerShown: false }} />
      <MainStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier mon profil" }} />
      <MainStack.Screen
        component={EditPreferencesScreen}
        name="EditPreferences"
        options={{ title: "Mes préférences" }}
      />
      <MainStack.Screen component={PrivacyScreen} name="Privacy" options={{ title: "Confidentialité" }} />
      <MainStack.Screen
        component={BlockedUsersScreen}
        name="BlockedUsers"
        options={{ title: "Utilisateurs bloqués" }}
      />
      <MainStack.Screen component={ChangePasswordScreen} name="ChangePassword" options={{ title: "Mot de passe" }} />
      <MainStack.Screen
        component={DeleteAccountScreen}
        name="DeleteAccount"
        options={{ title: "Supprimer mon compte" }}
      />
    </MainStack.Navigator>
  );
}
