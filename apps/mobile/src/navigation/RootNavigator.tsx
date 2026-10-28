import React from "react";
import { Pressable, View } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, useNavigation } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator, type NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import { Badge, Button, ErrorState, LoadingState, Screen, Text } from "../components";
import { useUnreadMessages, useUnreadNotifications } from "../hooks/useBadges";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import { ChatScreen } from "../screens/chats/ChatScreen";
import { ChatsScreen } from "../screens/chats/ChatsScreen";
import { DiscoverScreen } from "../screens/discover/DiscoverScreen";
import { ForgotPasswordScreen } from "../screens/auth/ForgotPasswordScreen";
import { LoginScreen } from "../screens/auth/LoginScreen";
import { NotificationsScreen } from "../screens/NotificationsScreen";
import { OnboardingScreen } from "../screens/onboarding/OnboardingScreen";
import { ProfileDetailScreen } from "../screens/ProfileDetailScreen";
import { BlockedUsersScreen } from "../screens/profile/BlockedUsersScreen";
import { ChangePasswordScreen } from "../screens/profile/ChangePasswordScreen";
import { DeleteAccountScreen } from "../screens/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../screens/profile/EditProfileScreen";
import { PhotosScreen } from "../screens/profile/PhotosScreen";
import { PreferencesScreen } from "../screens/profile/PreferencesScreen";
import { PrivacyScreen } from "../screens/profile/PrivacyScreen";
import { ProfileScreen } from "../screens/profile/ProfileScreen";
import { RegisterScreen } from "../screens/auth/RegisterScreen";
import { ReportScreen } from "../screens/ReportScreen";
import { ResetPasswordScreen } from "../screens/auth/ResetPasswordScreen";
import { WelcomeScreen } from "../screens/auth/WelcomeScreen";
import { MIN_TOUCH, useTheme } from "../theme";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function BellButton() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const unread = useUnreadNotifications();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={unread > 0 ? `Notifications, ${unread} unread` : "Notifications"}
      onPress={() => navigation.navigate("Notifications")}
      testID="header-bell"
      style={{ minWidth: MIN_TOUCH, minHeight: MIN_TOUCH, alignItems: "center", justifyContent: "center", marginRight: 4 }}
    >
      <Text style={{ fontSize: 22 }}>🔔</Text>
      <View style={{ position: "absolute", top: 2, right: 2 }}>
        <Badge count={unread} testID="bell-badge" />
      </View>
    </Pressable>
  );
}

function TabIcon({ glyph, color }: { glyph: string; color: string }) {
  return <Text style={{ color, fontSize: 20 }}>{glyph}</Text>;
}

function MainTabs() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessages();
  return (
    <Tabs.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerShadowVisible: false,
        headerTitleStyle: { color: colors.text, fontWeight: "700" },
        headerRight: () => <BellButton />,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border }
      }}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ tabBarIcon: ({ color }) => <TabIcon glyph="✦" color={color} />, tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        name="Chats"
        component={ChatsScreen}
        options={{
          tabBarIcon: ({ color }) => <TabIcon glyph="✉" color={color} />,
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarTestID: "tab-chats"
        }}
      />
      <Tabs.Screen name="Profile" component={ProfileScreen} options={{ tabBarIcon: ({ color }) => <TabIcon glyph="☺" color={color} />, tabBarTestID: "tab-profile" }} />
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
          headerTintColor: colors.text,
          headerShadowVisible: false,
          contentStyle: { backgroundColor: colors.background }
        }}
      >
        <MainStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
        <MainStack.Screen name="Chat" component={ChatScreen} />
        <MainStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ title: "" }} />
        <MainStack.Screen name="Notifications" component={NotificationsScreen} />
        <MainStack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Edit profile" }} />
        <MainStack.Screen name="Photos" component={PhotosScreen} />
        <MainStack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Discovery preferences" }} />
        <MainStack.Screen name="Privacy" component={PrivacyScreen} />
        <MainStack.Screen name="BlockedUsers" component={BlockedUsersScreen} options={{ title: "Blocked users" }} />
        <MainStack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ title: "Change password" }} />
        <MainStack.Screen name="DeleteAccount" component={DeleteAccountScreen} options={{ title: "Delete account" }} />
        <MainStack.Screen name="Report" component={ReportScreen} options={{ presentation: "modal" }} />
      </MainStack.Navigator>
    </RealtimeProvider>
  );
}

function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { isDark, colors } = useTheme();
  const { user, isBooting, bootError, retryBoot, signOut } = useAuth();
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = { ...base, colors: { ...base.colors, background: colors.background, card: colors.background, text: colors.text, border: colors.border, primary: colors.primary } };

  let content: React.ReactNode;
  if (isBooting) {
    content = <Screen><LoadingState label="Loading" /></Screen>;
  } else if (bootError) {
    content = (
      <Screen>
        <ErrorState message="We could not reach the server." onRetry={retryBoot} />
        <Button title="Sign out" variant="ghost" onPress={() => void signOut()} />
      </Screen>
    );
  } else if (!user) {
    content = <AuthNavigator />;
  } else if (!user.profileComplete) {
    content = <OnboardingScreen />;
  } else {
    content = <MainNavigator />;
  }

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
