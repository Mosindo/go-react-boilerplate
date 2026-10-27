import React, { useEffect, useState } from "react";
import { Text as RNText } from "react-native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useMyProfile, useBadgeCounts } from "../state/queries";
import { RealtimeProvider } from "../state/realtime";
import { useSession } from "../state/session";
import { dark, usePalette } from "../theme";
import { Button } from "../ui/Button";
import { Screen } from "../ui/Screen";
import { ErrorState, Loading } from "../ui/states";
import { ActivityScreen } from "../screens/ActivityScreen";
import {
  BlockedScreen,
  ChangePasswordScreen,
  DeleteAccountScreen,
  EditProfileScreen,
  PhotosScreen,
  PreferencesScreen,
  ReportScreen,
  SettingsScreen
} from "../screens/AccountScreens";
import { ForgotPasswordScreen, LoginScreen, RegisterScreen } from "../screens/AuthScreens";
import { ChatScreen } from "../screens/ChatScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { MeScreen } from "../screens/MeScreen";
import { MessagesScreen } from "../screens/MessagesScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { UserProfileScreen } from "../screens/UserProfileScreen";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const TAB_ICONS: Record<keyof TabParamList, string> = {
  Discover: "🔥",
  Messages: "💬",
  Activity: "🔔",
  Me: "👤"
};

function TabsNavigator() {
  const p = usePalette();
  const badges = useBadgeCounts();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: p.primary,
        tabBarInactiveTintColor: p.textMuted,
        tabBarStyle: { backgroundColor: p.surface, borderTopColor: p.border },
        tabBarIcon: () => <RNText style={{ fontSize: 20 }}>{TAB_ICONS[route.name]}</RNText>
      })}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{ tabBarTestID: "tab-discover" }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{ tabBarBadge: badges.messages || undefined, tabBarTestID: "tab-messages" }}
      />
      <Tabs.Screen
        name="Activity"
        component={ActivityScreen}
        options={{ tabBarBadge: badges.activity || undefined, tabBarTestID: "tab-activity" }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{ title: "Profile", tabBarTestID: "tab-me" }}
      />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  const p = usePalette();
  return (
    <MainStack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: p.background },
        headerTintColor: p.text,
        headerShadowVisible: false,
        contentStyle: { backgroundColor: p.background }
      }}
    >
      <MainStack.Screen name="Tabs" component={TabsNavigator} options={{ headerShown: false }} />
      <MainStack.Screen name="Chat" component={ChatScreen} />
      <MainStack.Screen name="UserProfile" component={UserProfileScreen} options={{ title: "" }} />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: "Edit profile" }}
      />
      <MainStack.Screen name="Photos" component={PhotosScreen} />
      <MainStack.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{ title: "Discovery preferences" }}
      />
      <MainStack.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: "Privacy & account" }}
      />
      <MainStack.Screen
        name="Blocked"
        component={BlockedScreen}
        options={{ title: "Blocked people" }}
      />
      <MainStack.Screen
        name="ChangePassword"
        component={ChangePasswordScreen}
        options={{ title: "Change password" }}
      />
      <MainStack.Screen
        name="DeleteAccount"
        component={DeleteAccountScreen}
        options={{ title: "Delete account" }}
      />
      <MainStack.Screen
        name="Report"
        component={ReportScreen}
        options={{ presentation: "modal" }}
      />
    </MainStack.Navigator>
  );
}

/** Signed-in gate: no profile or no photo yet means onboarding first. */
function SignedIn() {
  const query = useMyProfile();
  const profile = query.data ?? null;
  const [onboarding, setOnboarding] = useState<boolean | null>(null);

  useEffect(() => {
    if (onboarding === null && query.isSuccess) setOnboarding(!profile?.complete);
  }, [onboarding, query.isSuccess, profile?.complete]);

  if (query.isError) {
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  if (onboarding === null) {
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  }
  if (onboarding)
    return <OnboardingScreen profile={profile} onFinish={() => setOnboarding(false)} />;
  return <MainNavigator />;
}

function AuthNavigator() {
  const p = usePalette();
  return (
    <AuthStack.Navigator
      screenOptions={{ headerShown: false, contentStyle: { backgroundColor: p.background } }}
    >
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status, retryBoot } = useSession();
  const p = usePalette();
  const isDark = p === dark;
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: {
      ...base.colors,
      background: p.background,
      card: p.surface,
      text: p.text,
      border: p.border,
      primary: p.primary
    }
  };

  let content: React.ReactNode;
  if (status === "booting") {
    content = (
      <Screen>
        <Loading label="Starting Ember…" />
      </Screen>
    );
  } else if (status === "unreachable") {
    content = (
      <Screen>
        <ErrorState
          message="Cannot reach the server. Check your connection and the API address, then try again."
          onRetry={retryBoot}
        />
        <Button label="Retry" variant="ghost" onPress={retryBoot} />
      </Screen>
    );
  } else if (status === "signedOut") {
    content = <AuthNavigator />;
  } else {
    content = (
      <RealtimeProvider>
        <SignedIn />
      </RealtimeProvider>
    );
  }
  return <NavigationContainer theme={theme}>{content}</NavigationContainer>;
}
