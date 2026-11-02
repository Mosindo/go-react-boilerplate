import React, { useEffect, useState } from "react";
import { Text } from "react-native";
import { NavigationContainer, DefaultTheme, DarkTheme } from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { StatusBar } from "expo-status-bar";
import { useAuth } from "../auth/AuthContext";
import { Badge, ErrorState, Loading } from "../components/ui";
import { useProfileBundle, useUnreadMessages, useUnreadNotifications } from "../hooks";
import { apiErrorMessage } from "../api/client";
import { ForgotScreen, LoginScreen, RegisterScreen, WelcomeScreen } from "../screens/auth/AuthScreens";
import DiscoverScreen from "../screens/discover/DiscoverScreen";
import UserProfileScreen from "../screens/discover/UserProfileScreen";
import ChatScreen from "../screens/matches/ChatScreen";
import MatchesScreen from "../screens/matches/MatchesScreen";
import NotificationsScreen from "../screens/matches/NotificationsScreen";
import OnboardingScreen from "../screens/onboarding/OnboardingScreen";
import {
  BlockedScreen,
  EditProfileScreen,
  MeScreen,
  PhotosScreen,
  PreferencesScreen,
  PrivacyScreen
} from "../screens/profile/ProfileScreens";
import { useTheme } from "../theme/theme";
import type { AuthStackParamList, MainStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function TabIcon({ glyph, color, count }: { glyph: string; color: string; count?: number }) {
  return (
    <>
      <Text style={{ fontSize: 22, color }}>{glyph}</Text>
      {count ? (
        <Text style={{ position: "absolute", top: -4, right: -12 }}>
          <Badge count={count} />
        </Text>
      ) : null}
    </>
  );
}

function MainTabs() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessages();
  const unreadNotifications = useUnreadNotifications();
  return (
    <Tabs.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border }
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{
          title: "Découvrir",
          tabBarIcon: ({ color }) => <TabIcon glyph="🔥" color={color} />,
          tabBarAccessibilityLabel: "Découvrir"
        }}
      />
      <Tabs.Screen
        name="Matches"
        component={MatchesScreen}
        options={{
          title: "Messages",
          tabBarIcon: ({ color }) => <TabIcon glyph="💬" color={color} count={unreadMessages} />,
          tabBarAccessibilityLabel: `Messages${unreadMessages ? `, ${unreadMessages} non lus` : ""}`
        }}
      />
      <Tabs.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{
          title: "Alertes",
          tabBarIcon: ({ color }) => <TabIcon glyph="🔔" color={color} count={unreadNotifications} />,
          tabBarAccessibilityLabel: `Alertes${unreadNotifications ? `, ${unreadNotifications} non lues` : ""}`
        }}
      />
      <Tabs.Screen
        name="Me"
        component={MeScreen}
        options={{ title: "Moi", tabBarIcon: ({ color }) => <TabIcon glyph="👤" color={color} />, tabBarAccessibilityLabel: "Mon profil" }}
      />
    </Tabs.Navigator>
  );
}

function MainApp() {
  const { colors } = useTheme();
  const options = {
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour"
  };
  return (
    <MainStack.Navigator screenOptions={options}>
      <MainStack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
      <MainStack.Screen name="Chat" component={ChatScreen} />
      <MainStack.Screen name="UserProfile" component={UserProfileScreen} options={{ title: "Profil" }} />
      <MainStack.Screen name="EditProfile" component={EditProfileScreen} options={{ title: "Modifier mon profil" }} />
      <MainStack.Screen name="Preferences" component={PreferencesScreen} options={{ title: "Préférences" }} />
      <MainStack.Screen name="Photos" component={PhotosScreen} options={{ title: "Mes photos" }} />
      <MainStack.Screen name="Privacy" component={PrivacyScreen} options={{ title: "Confidentialité" }} />
      <MainStack.Screen name="Blocked" component={BlockedScreen} options={{ title: "Personnes bloquées" }} />
    </MainStack.Navigator>
  );
}

function SignedIn() {
  const bundle = useProfileBundle();
  // Once onboarding starts it stays on screen until the user taps "Terminer",
  // even though the profile becomes complete as soon as the first photo is added.
  const [inOnboarding, setInOnboarding] = useState<boolean | null>(null);
  useEffect(() => {
    if (inOnboarding === null && bundle.data) setInOnboarding(!bundle.data.complete);
  }, [bundle.data, inOnboarding]);

  if (bundle.isLoading || (bundle.data && inOnboarding === null)) return <Loading label="Chargement de votre profil…" />;
  if (bundle.isError) return <ErrorState message={apiErrorMessage(bundle.error)} onRetry={() => void bundle.refetch()} />;
  if (inOnboarding || !bundle.data?.complete) return <OnboardingScreen onFinish={() => setInOnboarding(false)} />;
  return <MainApp />;
}

export default function RootNavigator() {
  const { status, retryBoot } = useAuth();
  const { dark, colors } = useTheme();
  const base = dark ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      text: colors.text,
      border: colors.border,
      primary: colors.primary
    }
  };

  let body: React.ReactNode;
  if (status === "loading") body = <Loading label="Tandem" />;
  else if (status === "unreachable") {
    body = <ErrorState message="Impossible de joindre le serveur. Vérifiez votre connexion." onRetry={retryBoot} />;
  } else if (status === "out") {
    body = (
      <AuthStack.Navigator
        screenOptions={{
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text,
          headerShadowVisible: false,
          headerTitle: "",
          headerBackTitle: "Retour"
        }}
      >
        <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
        <AuthStack.Screen name="Login" component={LoginScreen} />
        <AuthStack.Screen name="Register" component={RegisterScreen} />
        <AuthStack.Screen name="Forgot" component={ForgotScreen} />
      </AuthStack.Navigator>
    );
  } else body = <SignedIn />;

  return (
    <NavigationContainer theme={theme}>
      <StatusBar style={dark ? "light" : "dark"} />
      {body}
    </NavigationContainer>
  );
}
