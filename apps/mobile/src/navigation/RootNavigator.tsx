import { Ionicons } from "@expo/vector-icons";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React, { createContext, useContext, useEffect, useMemo, useState } from "react";

import { useAuth } from "../auth/AuthContext";
import { useProfileStatus, useSummary } from "../hooks";
import {
  ForgotPasswordScreen,
  LoginScreen,
  RegisterScreen,
  ResetPasswordScreen,
  WelcomeScreen,
} from "../screens/auth/AuthScreens";
import { ChatScreen } from "../screens/chat/ChatScreen";
import { MessagesScreen } from "../screens/chat/MessagesScreen";
import { DiscoverScreen } from "../screens/discover/DiscoverScreen";
import { ProfileDetailScreen } from "../screens/discover/ProfileDetailScreen";
import { NotificationsScreen } from "../screens/notifications/NotificationsScreen";
import { PreferencesSetupScreen } from "../screens/profile/PreferencesScreen";
import { ProfileSetupScreen } from "../screens/profile/ProfileFormScreen";
import {
  EditProfileScreen,
  PhotosScreen,
  PhotosSetupScreen,
  PreferencesScreen,
} from "../screens/profile/ProfileScreens";
import { MeScreen } from "../screens/settings/MeScreen";
import {
  BlockedScreen,
  ChangePasswordScreen,
  DeleteAccountScreen,
  SettingsScreen,
} from "../screens/settings/SettingsScreens";
import { useTheme } from "../theme";
import { ErrorState, Loading } from "../ui/States";
import type {
  AuthStackParamList,
  MainStackParamList,
  OnboardingParamList,
  TabParamList,
} from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const OnboardingStack = createNativeStackNavigator<OnboardingParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

/**
 * Onboarding stays mounted until the person taps "Terminer", even though the profile becomes
 * "complete" server-side as soon as the first photo is uploaded.
 */
const OnboardingContext = createContext<{ finish: () => void }>({ finish: () => undefined });

function AuthNavigator() {
  return (
    <AuthStack.Navigator
      screenOptions={{ headerShadowVisible: false, headerTitle: "", headerBackTitle: "Retour" }}
    >
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
    </AuthStack.Navigator>
  );
}

function OnboardingNavigator({ hasProfile }: { hasProfile: boolean }) {
  const { finish } = useContext(OnboardingContext);
  const status = useProfileStatus();
  return (
    <OnboardingStack.Navigator
      initialRouteName={hasProfile ? "PhotosSetup" : "ProfileSetup"}
      screenOptions={{
        headerShadowVisible: false,
        headerTitle: "",
        headerBackVisible: false,
        gestureEnabled: false,
      }}
    >
      <OnboardingStack.Screen name="ProfileSetup">
        {({ navigation }) => (
          <ProfileSetupScreen onDone={() => navigation.navigate("PreferencesSetup")} />
        )}
      </OnboardingStack.Screen>
      <OnboardingStack.Screen name="PreferencesSetup">
        {({ navigation }) => (
          <PreferencesSetupScreen
            initial={status.data?.preferences ?? null}
            onDone={() => navigation.navigate("PhotosSetup")}
          />
        )}
      </OnboardingStack.Screen>
      <OnboardingStack.Screen name="PhotosSetup">
        {() => <PhotosSetupScreen onDone={finish} />}
      </OnboardingStack.Screen>
    </OnboardingStack.Navigator>
  );
}

function MainTabs() {
  const { colors } = useTheme();
  const summary = useSummary();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        tabBarIcon: ({ color, size, focused }) => {
          const icons = {
            Discover: focused ? "flame" : "flame-outline",
            Messages: focused ? "chatbubbles" : "chatbubbles-outline",
            Alerts: focused ? "notifications" : "notifications-outline",
            Me: focused ? "person" : "person-outline",
          } as const;
          return <Ionicons name={icons[route.name]} size={size} color={color} />;
        },
      })}
    >
      <Tabs.Screen name="Discover" component={DiscoverScreen} options={{ title: "Découvrir" }} />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{ title: "Messages", tabBarBadge: summary.data?.unreadMessages || undefined }}
      />
      <Tabs.Screen
        name="Alerts"
        component={NotificationsScreen}
        options={{ title: "Alertes", tabBarBadge: summary.data?.unreadNotifications || undefined }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil" }} />
    </Tabs.Navigator>
  );
}

function MainNavigator() {
  return (
    <MainStack.Navigator screenOptions={{ headerShadowVisible: false, headerBackTitle: "Retour" }}>
      <MainStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
      <MainStack.Screen name="Chat" component={ChatScreen} />
      <MainStack.Screen name="ProfileDetail" component={ProfileDetailScreen} />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: "Mes informations" }}
      />
      <MainStack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
      <MainStack.Screen
        name="Preferences"
        component={PreferencesScreen}
        options={{ title: "Préférences" }}
      />
      <MainStack.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: "Réglages" }}
      />
      <MainStack.Screen
        name="Blocked"
        component={BlockedScreen}
        options={{ title: "Personnes bloquées" }}
      />
      <MainStack.Screen
        name="ChangePassword"
        component={ChangePasswordScreen}
        options={{ title: "Mot de passe" }}
      />
      <MainStack.Screen
        name="DeleteAccount"
        component={DeleteAccountScreen}
        options={{ title: "" }}
      />
    </MainStack.Navigator>
  );
}

function SignedIn() {
  const status = useProfileStatus();
  // null until the first server answer: the session is an onboarding session if the profile was incomplete then.
  const [onboarding, setOnboarding] = useState<boolean | null>(null);
  const ctx = useMemo(() => ({ finish: () => setOnboarding(false) }), []);
  const complete = status.data?.complete;

  useEffect(() => {
    if (onboarding === null && complete !== undefined) setOnboarding(!complete);
  }, [onboarding, complete]);

  if (status.isLoading || (status.data && onboarding === null))
    return <Loading label="Chargement…" />;
  if (status.isError || !status.data)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;

  return (
    <OnboardingContext.Provider value={ctx}>
      {onboarding ? (
        <OnboardingNavigator hasProfile={status.data.profile !== null} />
      ) : (
        <MainNavigator />
      )}
    </OnboardingContext.Provider>
  );
}

export function RootNavigator() {
  const { status } = useAuth();
  const { colors, isDark } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  const theme = useMemo(
    () => ({
      ...base,
      colors: {
        ...base.colors,
        background: colors.background,
        card: colors.background,
        text: colors.text,
        border: colors.border,
        primary: colors.primary,
      },
    }),
    [base, colors],
  );

  if (status === "booting") return <Loading label="Restauration de la session…" />;
  return (
    <NavigationContainer theme={theme}>
      {status === "authenticated" ? <SignedIn /> : <AuthNavigator />}
    </NavigationContainer>
  );
}
