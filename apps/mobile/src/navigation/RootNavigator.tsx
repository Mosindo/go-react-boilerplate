import React from "react";
import { Platform } from "react-native";
import { DarkTheme, NavigationContainer, type Theme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { errorMessage } from "../api/client";
import { useChatUnread, useMyProfile, useNotificationsUnread } from "../hooks/queries";
import { useRealtimeSync } from "../hooks/useRealtime";
import { useSession } from "../hooks/useSession";
import { ForgotPasswordScreen } from "../screens/auth/ForgotPasswordScreen";
import { LoginScreen } from "../screens/auth/LoginScreen";
import { RegisterScreen } from "../screens/auth/RegisterScreen";
import { ResetPasswordScreen } from "../screens/auth/ResetPasswordScreen";
import { WelcomeScreen } from "../screens/auth/WelcomeScreen";
import { ChatScreen } from "../screens/chat/ChatScreen";
import { MatchesScreen } from "../screens/chat/MatchesScreen";
import { DiscoverScreen } from "../screens/discover/DiscoverScreen";
import { ActivityScreen } from "../screens/notifications/ActivityScreen";
import { OnboardingScreen } from "../screens/onboarding/OnboardingScreen";
import { AccountScreen } from "../screens/profile/AccountScreen";
import { BlockedUsersScreen } from "../screens/profile/BlockedUsersScreen";
import { EditPhotosScreen } from "../screens/profile/EditPhotosScreen";
import { EditProfileScreen } from "../screens/profile/EditProfileScreen";
import { MeScreen } from "../screens/profile/MeScreen";
import { PreferencesScreen } from "../screens/profile/PreferencesScreen";
import { PrivacyScreen } from "../screens/profile/PrivacyScreen";
import { ProfileDetailScreen } from "../screens/profile/ProfileDetailScreen";
import { colors } from "../theme";
import { ErrorState, LoadingState, Screen } from "../ui";
import type { AppStackParamList, AuthStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const AppStack = createNativeStackNavigator<AppStackParamList>();
const Tab = createBottomTabNavigator<TabParamList>();

const navigationTheme: Theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    primary: colors.primary,
    background: colors.background,
    card: colors.backgroundElevated,
    text: colors.text,
    border: colors.border,
    notification: colors.primary
  }
};

const stackOptions = { headerShown: false, contentStyle: { backgroundColor: colors.background }, animation: "slide_from_right" as const };

function Tabs() {
  const chatUnread = useChatUnread();
  const notificationsUnread = useNotificationsUnread();
  const badge = (n: number | undefined) => (n ? (n > 99 ? "99+" : n) : undefined);
  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: {
          backgroundColor: colors.backgroundElevated,
          borderTopColor: colors.border,
          height: Platform.OS === "web" ? 64 : undefined
        },
        tabBarLabelStyle: { fontSize: 12, fontWeight: "600" },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ color, focused, size }) => {
          const icons: Record<keyof TabParamList, [keyof typeof Ionicons.glyphMap, keyof typeof Ionicons.glyphMap]> = {
            Discover: ["flame", "flame-outline"],
            Matches: ["chatbubbles", "chatbubbles-outline"],
            Activity: ["notifications", "notifications-outline"],
            Me: ["person-circle", "person-circle-outline"]
          };
          const [active, inactive] = icons[route.name];
          return <Ionicons name={focused ? active : inactive} color={color} size={size} />;
        }
      })}
      sceneContainerStyle={{ backgroundColor: colors.background }}
    >
      <Tab.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir", tabBarTestID: "tab-discover" }} />
      <Tab.Screen
        name="Matches"
        component={MatchesScreen}
        options={{ title: "Matchs", tabBarBadge: badge(chatUnread.data), tabBarTestID: "tab-matches" }}
      />
      <Tab.Screen
        name="Activity"
        component={ActivityScreen}
        options={{ title: "Activité", tabBarBadge: badge(notificationsUnread.data), tabBarTestID: "tab-activity" }}
      />
      <Tab.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tab.Navigator>
  );
}

function SignedIn() {
  const { user } = useSession();
  const me = useMyProfile();
  const complete = me.data?.completeness.isComplete ?? false;
  useRealtimeSync(complete, user?.id);

  if (me.isLoading) {
    return (
      <Screen>
        <LoadingState label="Chargement de votre profil…" />
      </Screen>
    );
  }
  if (!me.data) {
    return (
      <Screen>
        <ErrorState message={errorMessage(me.error)} onRetry={() => void me.refetch()} />
      </Screen>
    );
  }
  if (!complete) {
    return <OnboardingScreen me={me.data} />;
  }
  return (
    <AppStack.Navigator screenOptions={stackOptions}>
      <AppStack.Screen name="Tabs" component={Tabs} />
      <AppStack.Screen name="Chat" component={ChatScreen} />
      <AppStack.Screen name="ProfileDetail" component={ProfileDetailScreen} options={{ animation: "slide_from_bottom" }} />
      <AppStack.Screen name="EditProfile" component={EditProfileScreen} />
      <AppStack.Screen name="EditPhotos" component={EditPhotosScreen} />
      <AppStack.Screen name="Preferences" component={PreferencesScreen} />
      <AppStack.Screen name="Privacy" component={PrivacyScreen} />
      <AppStack.Screen name="BlockedUsers" component={BlockedUsersScreen} />
      <AppStack.Screen name="Account" component={AccountScreen} />
    </AppStack.Navigator>
  );
}

function SignedOut() {
  return (
    <AuthStack.Navigator screenOptions={stackOptions}>
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
    </AuthStack.Navigator>
  );
}

export function RootNavigator() {
  const { status } = useSession();
  return (
    <NavigationContainer theme={navigationTheme} documentTitle={{ formatter: () => "Lueur" }}>
      {status === "booting" ? (
        <Screen>
          <LoadingState />
        </Screen>
      ) : status === "signedIn" ? (
        <SignedIn />
      ) : (
        <SignedOut />
      )}
    </NavigationContainer>
  );
}
