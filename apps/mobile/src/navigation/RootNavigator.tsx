import React, { createContext, useContext, useEffect, useState } from "react";
import { Ionicons } from "@expo/vector-icons";
import {
  DarkTheme,
  DefaultTheme,
  NavigationContainer,
  type Theme as NavTheme
} from "@react-navigation/native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import {
  ForgotPasswordScreen,
  LoginScreen,
  RegisterScreen,
  WelcomeScreen
} from "../features/auth/AuthScreens";
import { ChatScreen } from "../features/chat/ChatScreen";
import { MessagesScreen } from "../features/chat/MessagesScreen";
import { DiscoverScreen } from "../features/discover/DiscoverScreen";
import { NotificationsScreen } from "../features/notifications/NotificationsScreen";
import { EditProfileScreen } from "../features/profile/EditProfileScreen";
import { MeScreen } from "../features/profile/MeScreen";
import { OnboardingScreen } from "../features/profile/OnboardingScreen";
import { PreferencesScreen, PrivacyScreen, SecurityScreen } from "../features/profile/SettingsScreens";
import { UserProfileScreen } from "../features/profile/UserProfileScreen";
import { useAuth } from "../auth/AuthContext";
import { useMyProfile, useUnreadMessages } from "../hooks/queries";
import { useTheme } from "../theme/theme";
import { ErrorState, LoadingView } from "../ui/Feedback";
import { Screen } from "../ui/Screen";
import { navigationRef } from "./ref";
import type { RootParamList, TabParamList } from "./types";

const Stack = createNativeStackNavigator<RootParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const ICONS: Record<
  keyof TabParamList,
  { on: keyof typeof Ionicons.glyphMap; off: keyof typeof Ionicons.glyphMap }
> = {
  Discover: { on: "flame", off: "flame-outline" },
  Messages: { on: "chatbubbles", off: "chatbubbles-outline" },
  Me: { on: "person", off: "person-outline" }
};

function MainTabs() {
  const { colors } = useTheme();
  const unread = useUnreadMessages();
  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textSubtle,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarIcon: ({ focused, color, size }) => (
          <Ionicons name={focused ? ICONS[route.name].on : ICONS[route.name].off} size={size} color={color} />
        )
      })}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{ title: "Découvrir", tabBarTestID: "tab-discover" }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: "Messages",
          tabBarBadge: unread.data && unread.data > 0 ? unread.data : undefined,
          tabBarTestID: "tab-messages"
        }}
      />
      <Tabs.Screen name="Me" component={MeScreen} options={{ title: "Profil", tabBarTestID: "tab-me" }} />
    </Tabs.Navigator>
  );
}

const OnboardingContext = createContext<{ finish: () => void }>({ finish: () => undefined });
export const useOnboarding = () => useContext(OnboardingContext);

function SignedIn() {
  const profile = useMyProfile();
  // Onboarding stays on screen until the user finishes its last step, even though the profile
  // already counts as "complete" (and so visible in discovery) once photo and location are set.
  const [onboarding, setOnboarding] = useState<boolean | null>(null);
  const complete = profile.data?.complete;
  useEffect(() => {
    if (complete === undefined) return;
    setOnboarding((current) => (current === null ? !complete : complete ? current : true));
  }, [complete]);
  const { colors } = useTheme();
  const headerOptions = {
    headerStyle: { backgroundColor: colors.background },
    headerTintColor: colors.text,
    headerShadowVisible: false,
    headerBackTitle: "Retour"
  } as const;

  if (profile.isPending)
    return (
      <Screen>
        <LoadingView label="Chargement de ton profil…" />
      </Screen>
    );
  if (profile.isError)
    return (
      <Screen>
        <ErrorState message={profile.error.message} onRetry={() => void profile.refetch()} />
      </Screen>
    );

  if (onboarding === null)
    return (
      <Screen>
        <LoadingView />
      </Screen>
    );

  return (
    <OnboardingContext.Provider value={{ finish: () => setOnboarding(false) }}>
      <Stack.Navigator screenOptions={headerOptions}>
        {!onboarding ? (
          <>
            <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
            <Stack.Screen name="Chat" component={ChatScreen} options={{ headerShown: false }} />
            <Stack.Screen
              name="UserProfile"
              component={UserProfileScreen}
              options={{ title: "", headerTransparent: false }}
            />
            <Stack.Screen
              name="Notifications"
              component={NotificationsScreen}
              options={{ title: "Notifications" }}
            />
            <Stack.Screen
              name="EditProfile"
              component={EditProfileScreen}
              options={{ title: "Mon profil" }}
            />
            <Stack.Screen
              name="Preferences"
              component={PreferencesScreen}
              options={{ title: "Préférences" }}
            />
            <Stack.Screen name="Privacy" component={PrivacyScreen} options={{ title: "Confidentialité" }} />
            <Stack.Screen name="Security" component={SecurityScreen} options={{ title: "Sécurité" }} />
          </>
        ) : (
          <Stack.Screen name="Onboarding" component={OnboardingScreen} options={{ headerShown: false }} />
        )}
      </Stack.Navigator>
    </OnboardingContext.Provider>
  );
}

function SignedOut() {
  const { colors } = useTheme();
  return (
    <Stack.Navigator
      screenOptions={{
        headerStyle: { backgroundColor: colors.background },
        headerTintColor: colors.text,
        headerShadowVisible: false,
        title: "",
        headerBackTitle: "Retour"
      }}
    >
      <Stack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <Stack.Screen name="Login" component={LoginScreen} />
      <Stack.Screen name="Register" component={RegisterScreen} />
      <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
    </Stack.Navigator>
  );
}

export function RootNavigator() {
  const { status, bootError, retryBoot } = useAuth();
  const { dark, colors } = useTheme();
  const base = dark ? DarkTheme : DefaultTheme;
  const navTheme: NavTheme = {
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

  let content: React.ReactNode;
  if (bootError)
    content = (
      <Screen>
        <ErrorState message={bootError} onRetry={retryBoot} />
      </Screen>
    );
  else if (status === "booting")
    content = (
      <Screen>
        <LoadingView />
      </Screen>
    );
  else content = status === "signedIn" ? <SignedIn /> : <SignedOut />;

  return (
    <NavigationContainer ref={navigationRef} theme={navTheme}>
      {content}
    </NavigationContainer>
  );
}
