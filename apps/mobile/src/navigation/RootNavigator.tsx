import React from "react";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createBottomTabNavigator, type BottomTabBarProps } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { errorMessage } from "../api/client";
import { RealtimeBridge } from "../hooks/realtime";
import { useConversations, useNotifications, useOwnProfile } from "../hooks/queries";
import { useSession } from "../hooks/session";
import { ForgotScreen } from "../features/auth/ForgotScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { ChatScreen } from "../features/messages/ChatScreen";
import { ConversationsScreen } from "../features/messages/ConversationsScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../features/discover/ProfileDetailScreen";
import { LikesScreen } from "../features/likes/LikesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { MeScreen } from "../features/settings/MeScreen";
import { BlockedScreen, ChangePasswordScreen, EditProfileScreen, PhotosScreen, PreferencesScreen } from "../features/settings/SettingsScreens";
import { Badge, ErrorState, Loader, Text, MIN_TOUCH, spacing, useColors, useIsDark } from "../shared/ui";
import type { AuthStackParams, MainStackParams, TabParams } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParams>();
const MainStack = createNativeStackNavigator<MainStackParams>();
const Tab = createBottomTabNavigator<TabParams>();

const TAB_LABELS: Record<keyof TabParams, string> = {
  Discover: "Découvrir",
  Likes: "Likes",
  Messages: "Messages",
  Notifications: "Alertes",
  Me: "Profil",
};

function TabBar({ state, navigation }: BottomTabBarProps) {
  const c = useColors();
  const insets = useSafeAreaInsets();
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = conversations.data?.pages.reduce((n, p) => n + p.conversations.reduce((m, cv) => m + cv.unread, 0), 0) ?? 0;
  const unreadNotifications = notifications.data?.pages[0]?.unread ?? 0;
  const badges: Partial<Record<keyof TabParams, number>> = { Messages: unreadMessages, Notifications: unreadNotifications };

  return (
    <View style={[styles.bar, { backgroundColor: c.surface, borderTopColor: c.border, paddingBottom: Math.max(insets.bottom, spacing.xs) }]}>
      {state.routes.map((route, index) => {
        const focused = state.index === index;
        const name = route.name as keyof TabParams;
        return (
          <Pressable
            key={route.key}
            accessibilityRole="tab"
            accessibilityState={{ selected: focused }}
            accessibilityLabel={TAB_LABELS[name]}
            testID={`tab-${name}`}
            onPress={() => {
              const event = navigation.emit({ type: "tabPress", target: route.key, canPreventDefault: true });
              if (!focused && !event.defaultPrevented) navigation.navigate(route.name);
            }}
            style={styles.tab}
          >
            <View style={[styles.indicator, { backgroundColor: focused ? c.accent : "transparent" }]} />
            <View style={{ flexDirection: "row", alignItems: "center", gap: 4 }}>
              <Text variant="label" style={{ color: focused ? c.primary : c.textMuted, fontWeight: focused ? "800" : "600" }}>{TAB_LABELS[name]}</Text>
              <Badge count={badges[name] ?? 0} />
            </View>
          </Pressable>
        );
      })}
    </View>
  );
}

function Tabs() {
  return (
    <Tab.Navigator tabBar={(props) => <TabBar {...props} />} screenOptions={{ headerShown: false }}>
      <Tab.Screen name="Discover" component={DiscoverScreen} />
      <Tab.Screen name="Likes" component={LikesScreen} />
      <Tab.Screen name="Messages" component={ConversationsScreen} />
      <Tab.Screen name="Notifications" component={NotificationsScreen} />
      <Tab.Screen name="Me" component={MeScreen} />
    </Tab.Navigator>
  );
}

function MainNavigator() {
  const c = useColors();
  const header = { headerShown: true, headerStyle: { backgroundColor: c.background }, headerTintColor: c.primary, headerTitleStyle: { color: c.text }, headerShadowVisible: false, headerBackTitle: "Retour" } as const;
  return (
    <>
      <RealtimeBridge />
      <MainStack.Navigator screenOptions={{ headerShown: false }}>
        <MainStack.Screen name="Tabs" component={Tabs} />
        <MainStack.Screen name="Chat" component={ChatScreen} options={header} />
        <MainStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ ...header, title: "Profil" }} />
        <MainStack.Screen name="EditProfile" component={EditProfileScreen} options={{ ...header, title: "Modifier mon profil" }} />
        <MainStack.Screen name="Photos" component={PhotosScreen} options={{ ...header, title: "Mes photos" }} />
        <MainStack.Screen name="Preferences" component={PreferencesScreen} options={{ ...header, title: "Préférences" }} />
        <MainStack.Screen name="Blocked" component={BlockedScreen} options={{ ...header, title: "Personnes bloquées" }} />
        <MainStack.Screen name="ChangePassword" component={ChangePasswordScreen} options={{ ...header, title: "Mot de passe" }} />
      </MainStack.Navigator>
    </>
  );
}

/** Signed-in gate: onboarding until the server reports the profile as complete. */
function SignedIn() {
  const profile = useOwnProfile();
  if (profile.isPending) return <Loader label="Chargement…" />;
  if (profile.isError) return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  if (!profile.data.isComplete) return <OnboardingScreen profile={profile.data} />;
  return <MainNavigator />;
}

function AuthNavigator() {
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false }}>
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="Forgot" component={ForgotScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, retryBoot } = useSession();
  const c = useColors();
  const dark = useIsDark();
  const base = dark ? DarkTheme : DefaultTheme;
  const theme = { ...base, colors: { ...base.colors, background: c.background, card: c.surface, text: c.text, border: c.border, primary: c.primary, notification: c.accent } };

  let content: React.ReactNode;
  if (status === "booting") content = <Loader label="Restauration de la session…" />;
  else if (status === "bootError") content = <ErrorState message="Impossible de joindre le serveur." onRetry={retryBoot} />;
  else if (status === "signedOut") content = <AuthNavigator />;
  else content = <SignedIn />;

  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}

const styles = StyleSheet.create({
  bar: { flexDirection: "row", borderTopWidth: StyleSheet.hairlineWidth },
  tab: { flex: 1, minHeight: MIN_TOUCH + 8, alignItems: "center", justifyContent: "center", gap: 4 },
  indicator: { width: 24, height: 3, borderRadius: 2 },
});
