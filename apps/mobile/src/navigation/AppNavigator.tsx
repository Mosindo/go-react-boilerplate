import React from "react";
import { View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import {
  DarkTheme,
  DefaultTheme,
  NavigationContainer,
  type Theme as NavTheme
} from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { t } from "../i18n/fr";
import { useUnreadBadges } from "../queries";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import ForgotPasswordScreen from "../screens/auth/ForgotPasswordScreen";
import LoginScreen from "../screens/auth/LoginScreen";
import RegisterScreen from "../screens/auth/RegisterScreen";
import ResetPasswordScreen from "../screens/auth/ResetPasswordScreen";
import ChatScreen from "../screens/main/ChatScreen";
import DiscoverScreen from "../screens/main/DiscoverScreen";
import MatchesScreen from "../screens/main/MatchesScreen";
import NotificationsScreen from "../screens/main/NotificationsScreen";
import ProfileDetailScreen from "../screens/main/ProfileDetailScreen";
import ProfileScreen from "../screens/main/ProfileScreen";
import type { AuthStackParams, MainStackParams, TabParams } from "../screens/navTypes";
import OnboardingScreen from "../screens/OnboardingScreen";
import BlockedScreen from "../screens/settings/BlockedScreen";
import DeleteAccountScreen from "../screens/settings/DeleteAccountScreen";
import EditProfileScreen from "../screens/settings/EditProfileScreen";
import ManagePhotosScreen from "../screens/settings/ManagePhotosScreen";
import PreferencesScreen from "../screens/settings/PreferencesScreen";
import PrivacyScreen from "../screens/settings/PrivacyScreen";
import { useSession } from "../session/SessionProvider";
import { useTheme } from "../theme/ThemeProvider";
import { ErrorState, LoadingState, Text } from "../ui";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tabs = createBottomTabNavigator<TabParams>();

const TAB_GLYPHS: Record<keyof TabParams, string> = {
  Discover: "✦",
  Matches: "✉",
  Notifications: "✺",
  Profile: "☺"
};

function TabsNavigator() {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const badges = useUnreadBadges();
  const badgeFor = (name: keyof TabParams): number | undefined => {
    const count =
      name === "Matches" ? badges.messages : name === "Notifications" ? badges.notifications : 0;
    return count > 0 ? count : undefined;
  };

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerStyle: { backgroundColor: colors.background },
        headerTitleStyle: { color: colors.text, fontWeight: "800" },
        headerShadowVisible: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.border,
          minHeight: 56 + insets.bottom
        },
        tabBarBadge: badgeFor(route.name),
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: colors.onAccent },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarIcon: ({ color }) => (
          <View accessibilityElementsHidden importantForAccessibility="no">
            <Text style={{ color, fontSize: 20 }}>{TAB_GLYPHS[route.name]}</Text>
          </View>
        )
      })}
    >
      <Tabs.Screen
        component={DiscoverScreen}
        name="Discover"
        options={{
          title: t.tabs.discover,
          tabBarTestID: "tab-discover",
          tabBarAccessibilityLabel: t.tabs.discover
        }}
      />
      <Tabs.Screen
        component={MatchesScreen}
        name="Matches"
        options={{
          title: t.tabs.matches,
          tabBarTestID: "tab-matches",
          tabBarAccessibilityLabel: t.tabs.matches
        }}
      />
      <Tabs.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          title: t.tabs.notifications,
          tabBarTestID: "tab-notifications",
          tabBarAccessibilityLabel: t.tabs.notifications
        }}
      />
      <Tabs.Screen
        component={ProfileScreen}
        name="Profile"
        options={{
          title: t.tabs.profile,
          tabBarTestID: "tab-profile",
          tabBarAccessibilityLabel: t.tabs.profile
        }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const { colors } = useTheme();
  return (
    <RealtimeProvider>
      <MainStack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: colors.background },
          headerTitleStyle: { color: colors.text },
          headerTintColor: colors.primary,
          headerShadowVisible: false,
          headerBackTitle: t.common.back,
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <MainStack.Screen component={TabsNavigator} name="Tabs" options={{ headerShown: false }} />
        <MainStack.Screen
          component={ChatScreen}
          name="Chat"
          options={{ title: t.chat.unknownUser }}
        />
        <MainStack.Screen
          component={ProfileDetailScreen}
          name="ProfileDetail"
          options={{ title: "" }}
        />
        <MainStack.Screen
          component={EditProfileScreen}
          name="EditProfile"
          options={{ title: t.profile.editTitle }}
        />
        <MainStack.Screen
          component={ManagePhotosScreen}
          name="ManagePhotos"
          options={{ title: t.photos.title }}
        />
        <MainStack.Screen
          component={PreferencesScreen}
          name="Preferences"
          options={{ title: t.preferences.title }}
        />
        <MainStack.Screen
          component={PrivacyScreen}
          name="Privacy"
          options={{ title: t.privacy.title }}
        />
        <MainStack.Screen
          component={BlockedScreen}
          name="Blocked"
          options={{ title: t.blocked.title }}
        />
        <MainStack.Screen
          component={DeleteAccountScreen}
          name="DeleteAccount"
          options={{ title: t.deleteAccount.title }}
        />
      </MainStack.Navigator>
    </RealtimeProvider>
  );
}

function AuthNavigator() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: colors.background },
        animation: "fade"
      }}
    >
      <AuthStack.Screen component={LoginScreen} name="Login" />
      <AuthStack.Screen component={RegisterScreen} name="Register" />
      <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" />
      <AuthStack.Screen component={ResetPasswordScreen} name="ResetPassword" />
    </AuthStack.Navigator>
  );
}

/** Auth -> onboarding -> main, decided from the session and GET /me. */
function Gate() {
  const { status, me } = useSession();
  if (status === "booting") return <LoadingState />;
  if (status === "signedOut") return <AuthNavigator />;
  if (me.isPending) return <LoadingState />;
  if (me.isError || !me.data) {
    return <ErrorState onRetry={() => me.refetch()} title={t.errors.network} />;
  }
  if (!me.data.onboarding.completed) return <OnboardingScreen />;
  return <MainNavigator />;
}

export function AppNavigator() {
  const { colors, scheme } = useTheme();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
    ...base,
    colors: {
      ...base.colors,
      primary: colors.primary,
      background: colors.background,
      card: colors.background,
      text: colors.text,
      border: colors.border,
      notification: colors.accent
    }
  };
  return (
    <NavigationContainer theme={navTheme}>
      <Gate />
    </NavigationContainer>
  );
}
