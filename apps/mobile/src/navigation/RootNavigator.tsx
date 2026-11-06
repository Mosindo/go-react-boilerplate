import React from "react";
import { Text as RNText, View } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, type LinkingOptions } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import { Badge, Loader, Screen } from "../components/ui";
import { fr } from "../i18n/fr";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { ForgotScreen, LoginScreen } from "../screens/AuthScreens";
import { MessagesScreen, useConversations } from "../screens/MessagesScreen";
import { NotificationsScreen, useNotifications } from "../screens/NotificationsScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { ProfileDetailScreen } from "../screens/ProfileDetailScreen";
import {
  ChangePasswordScreen,
  EditProfileScreen,
  PhotosScreen,
  PreferencesScreen,
  PrivacyScreen,
  ProfileScreen,
  useMyProfile
} from "../screens/ProfileScreens";
import { ErrorState } from "../components/ui";
import { useTheme } from "../theme";
import type { AuthStackParamList, RootStackParamList, TabParamList } from "./types";

const Tab = createBottomTabNavigator<TabParamList>();
const Stack = createNativeStackNavigator<RootStackParamList>();
const AuthStack = createNativeStackNavigator<AuthStackParamList>();

function TabIcon({ glyph, color, badge }: { glyph: string; color: string; badge?: number }) {
  return (
    <View>
      <RNText style={{ fontSize: 20, color }}>{glyph}</RNText>
      {badge ? (
        <View style={{ position: "absolute", top: -6, right: -12 }}>
          <Badge count={badge} />
        </View>
      ) : null}
    </View>
  );
}

function Tabs() {
  const { colors } = useTheme();
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = conversations.data?.pages[0]?.unreadTotal ?? 0;
  const unreadNotifications = notifications.data?.pages[0]?.unreadCount ?? 0;
  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" }
      }}
    >
      <Tab.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{ title: fr.tabs.discover, tabBarTestID: "tab-discover", tabBarIcon: ({ color }) => <TabIcon glyph="🔥" color={color} /> }}
      />
      <Tab.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: fr.tabs.messages,
          tabBarTestID: "tab-messages",
          tabBarAccessibilityLabel: unreadMessages ? `${fr.tabs.messages}, ${unreadMessages} non lus` : fr.tabs.messages,
          tabBarIcon: ({ color }) => <TabIcon glyph="💬" color={color} badge={unreadMessages} />
        }}
      />
      <Tab.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{
          title: fr.tabs.notifications,
          tabBarTestID: "tab-notifications",
          tabBarAccessibilityLabel: unreadNotifications ? `${fr.tabs.notifications}, ${unreadNotifications} non lues` : fr.tabs.notifications,
          tabBarIcon: ({ color }) => <TabIcon glyph="🔔" color={color} badge={unreadNotifications} />
        }}
      />
      <Tab.Screen
        name="Profile"
        component={ProfileScreen}
        options={{ title: fr.tabs.profile, tabBarTestID: "tab-profile", tabBarIcon: ({ color }) => <TabIcon glyph="👤" color={color} /> }}
      />
    </Tab.Navigator>
  );
}

function MainStack() {
  const { colors } = useTheme();
  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.primary,
        headerTitleStyle: { color: colors.text },
        headerShadowVisible: false,
        headerBackTitle: fr.common.back,
        contentStyle: { backgroundColor: colors.background }
      }}
    >
      <Stack.Screen name="Tabs" component={Tabs} options={{ headerShown: false }} />
      <Stack.Screen name="Chat" component={ChatScreen} options={{ title: "" }} />
      <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "" }} />
      <Stack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: fr.profile.edit }} />
      <Stack.Screen name="Photos" component={PhotosScreen} options={{ title: fr.profile.photos }} />
      <Stack.Screen name="Preferences" component={PreferencesScreen} options={{ title: fr.profile.preferences }} />
      <Stack.Screen name="Privacy" component={PrivacyScreen} options={{ title: fr.profile.privacy }} />
      <Stack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ title: fr.profile.password }} />
    </Stack.Navigator>
  );
}

/** Signed in: wait for the profile, send people without a complete profile to onboarding. */
function Authenticated() {
  const profile = useMyProfile();
  if (profile.isLoading) {
    return (
      <Screen>
        <Loader />
      </Screen>
    );
  }
  if (profile.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  if (!profile.data?.onboardingComplete) return <OnboardingScreen />;
  return (
    <RealtimeProvider>
      <MainStack />
    </RealtimeProvider>
  );
}

const linking: LinkingOptions<RootStackParamList> = { prefixes: ["lueur://"], config: { screens: { Tabs: "" } } };

export function RootNavigator() {
  const { status } = useAuth();
  const { colors, isDark } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: { ...base.colors, background: colors.background, card: colors.surface, text: colors.text, border: colors.border, primary: colors.primary }
  };

  if (status === "booting") {
    return (
      <Screen>
        <Loader />
      </Screen>
    );
  }

  return (
    <NavigationContainer theme={theme} linking={status === "authenticated" ? linking : undefined}>
      {status === "authenticated" ? (
        <Authenticated />
      ) : (
        <AuthStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
          <AuthStack.Screen name="Login" component={LoginScreen} />
          <AuthStack.Screen name="Forgot" component={ForgotScreen} />
        </AuthStack.Navigator>
      )}
    </NavigationContainer>
  );
}
