import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { DarkTheme, DefaultTheme, NavigationContainer, type Theme as NavTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { notificationsApi } from "../api/notifications";
import { queryKeys } from "../api/queryKeys";
import { ForgotPasswordScreen } from "../features/auth/ForgotPasswordScreen";
import { LoginScreen } from "../features/auth/LoginScreen";
import { RegisterScreen } from "../features/auth/RegisterScreen";
import { WelcomeScreen } from "../features/auth/WelcomeScreen";
import { ChatScreen } from "../features/chat/ChatScreen";
import { ConversationsScreen } from "../features/chat/ConversationsScreen";
import { DiscoveryScreen } from "../features/discovery/DiscoveryScreen";
import { ProfileDetailScreen } from "../features/discovery/ProfileDetailScreen";
import { LikesScreen } from "../features/likes/LikesScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { OnboardingScreen } from "../features/onboarding/OnboardingScreen";
import { BlockedUsersScreen } from "../features/profile/BlockedUsersScreen";
import { ChangePasswordScreen } from "../features/profile/ChangePasswordScreen";
import { DeleteAccountScreen } from "../features/profile/DeleteAccountScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { useMyProfile } from "../features/profile/hooks";
import { MyProfileScreen } from "../features/profile/MyProfileScreen";
import { PreferencesScreen } from "../features/profile/PreferencesScreen";
import { SettingsScreen } from "../features/profile/SettingsScreen";
import { errorMessage } from "../lib/errors";
import { useRealtimeConnected } from "../realtime/RealtimeProvider";
import { useSession } from "../session/SessionProvider";
import { useTheme } from "../theme";
import { Button, ErrorState, Icon, Loading, type IconName } from "../ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const tabIcons: Record<keyof TabParamList, [IconName, IconName]> = {
  Discover: ["flame", "flame-outline"],
  Likes: ["heart", "heart-outline"],
  Messages: ["chatbubbles", "chatbubbles-outline"],
  Me: ["person-circle", "person-circle-outline"]
};

function MainTabs() {
  const { colors } = useTheme();
  const connected = useRealtimeConnected();
  const badges = useQuery({ queryKey: queryKeys.badges, queryFn: notificationsApi.badges, refetchInterval: connected ? 60_000 : 20_000 });
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border, height: 64, paddingTop: 6, paddingBottom: 8 },
        tabBarLabelStyle: { fontSize: 11, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.accent, color: "#fff", fontSize: 10 },
        tabBarIcon: ({ focused, size }) => (
          <Icon name={tabIcons[route.name][focused ? 0 : 1]} size={size} color={focused ? "primary" : "textSubtle"} />
        )
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoveryScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{ title: "Likes", tabBarBadge: badges.data?.likesReceived || undefined, tabBarTestID: "tab-likes" }}
      />
      <Tabs.Screen
        name="Messages"
        component={ConversationsScreen}
        options={{ title: "Messages", tabBarBadge: badges.data?.unreadConversations || undefined, tabBarTestID: "tab-messages" }}
      />
      <Tabs.Screen name="Me" component={MyProfileScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

function SignedInApp() {
  const { colors } = useTheme();
  const { signOut } = useSession();
  const me = useMyProfile();
  const [onboarding, setOnboarding] = useState(false);
  const incomplete = me.data ? !me.data.completeness.isComplete : false;

  // Stay in onboarding until the user finishes it, even once the profile
  // becomes complete mid-way (after the first photo).
  useEffect(() => {
    if (incomplete) setOnboarding(true);
  }, [incomplete]);

  if (me.isPending) return <Loading />;
  if (me.isError) {
    return (
      <View style={{ flex: 1, backgroundColor: colors.background }}>
        <ErrorState message={errorMessage(me.error)} onRetry={() => void me.refetch()} />
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} style={{ marginBottom: 40 }} />
      </View>
    );
  }
  if (incomplete || onboarding) {
    return <OnboardingScreen me={me.data} onFinish={() => setOnboarding(false)} />;
  }

  return (
    <AppStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
      <AppStack.Screen name="Tabs" component={MainTabs} />
      <AppStack.Screen
        name="ProfileDetail"
        component={ProfileDetailScreen}
        options={{ presentation: "modal", animation: "slide_from_bottom" }}
      />
      <AppStack.Screen name="Chat" component={ChatScreen} />
      <AppStack.Screen name="Notifications" component={NotificationsScreen} />
      <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
      <AppStack.Screen name="Preferences" component={PreferencesScreen} />
      <AppStack.Screen name="Settings" component={SettingsScreen} />
      <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
      <AppStack.Screen name="ChangePassword" component={ChangePasswordScreen} />
      <AppStack.Screen name="DeleteAccount" component={DeleteAccountScreen} />
    </AppStack.Navigator>
  );
}

function AuthFlow() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.background } }}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { colors, isDark } = useTheme();
  const { status } = useSession();
  const base = isDark ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      border: colors.border,
      primary: colors.primary,
      text: colors.text,
      notification: colors.accent
    }
  };
  return (
    <NavigationContainer theme={navTheme} documentTitle={{ formatter: () => "Lueur" }}>
      {status === "booting" ? <Loading /> : status === "signedIn" ? <SignedInApp /> : <AuthFlow />}
    </NavigationContainer>
  );
}
