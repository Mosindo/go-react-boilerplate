import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { DefaultTheme, DarkTheme, NavigationContainer, type LinkingOptions } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { useAuth } from "../hooks/useAuth";
import { useMyProfile } from "../hooks/queries";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import AuthScreen from "../screens/AuthScreen";
import BlockedScreen from "../screens/BlockedScreen";
import ChatScreen from "../screens/ChatScreen";
import EditProfileScreen from "../screens/EditProfileScreen";
import ForgotPasswordScreen from "../screens/ForgotPasswordScreen";
import HomeScreen from "../screens/HomeScreen";
import MessagesScreen from "../screens/MessagesScreen";
import NotificationsScreen from "../screens/NotificationsScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import PreferencesScreen from "../screens/PreferencesScreen";
import ProfileDetailScreen from "../screens/ProfileDetailScreen";
import ProfileScreen from "../screens/ProfileScreen";
import ResetPasswordScreen from "../screens/ResetPasswordScreen";
import SettingsScreen from "../screens/SettingsScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { BottomNavigation } from "../shared/layout";
import { useTheme } from "../theme";
import type { RootStackParamList, TabParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

const linking: LinkingOptions<RootStackParamList> = {
  prefixes: ["lueur://", "http://localhost:8081"],
  config: { screens: { ResetPassword: "reset-password" } }
};

function MainTabs() {
  return (
    <Tabs.Navigator tabBar={(props) => <BottomNavigation {...props} />} screenOptions={{ headerShown: false }}>
      <Tabs.Screen name="Discover" component={HomeScreen} />
      <Tabs.Screen name="Messages" component={MessagesScreen} />
      <Tabs.Screen name="Me" component={ProfileScreen} />
    </Tabs.Navigator>
  );
}

/** Signed in: decides between onboarding and the app from the server-side completion flags. */
function AuthenticatedApp() {
  const { data, isLoading, isError, error, refetch } = useMyProfile();
  if (isLoading) return <LoadingView label="Chargement…" />;
  if (!data) return isError ? <ErrorView error={error} onRetry={() => void refetch()} /> : <LoadingView />;

  return (
    <RealtimeProvider>
      <Stack.Navigator screenOptions={{ headerShown: false }}>
        {data.completion.complete ? (
          <>
            <Stack.Screen name="Tabs" component={MainTabs} />
            <Stack.Screen name="ProfileDetail" component={ProfileDetailScreen} />
            <Stack.Screen name="Chat" component={ChatScreen} />
            <Stack.Screen name="Notifications" component={NotificationsScreen} />
            <Stack.Screen name="EditProfile" component={EditProfileScreen} />
            <Stack.Screen name="Preferences" component={PreferencesScreen} />
            <Stack.Screen name="Settings" component={SettingsScreen} />
            <Stack.Screen name="BlockedUsers" component={BlockedScreen} />
          </>
        ) : (
          <Stack.Screen name="Onboarding" component={OnboardingScreen} />
        )}
      </Stack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { status, retryBoot } = useAuth();
  const { colors, isDark } = useTheme();
  const base = isDark ? DarkTheme : DefaultTheme;
  const navTheme = {
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

  if (status === "booting") return <LoadingView label="Restauration de votre session…" />;
  if (status === "unreachable") return <ErrorView error={new Error("offline")} onRetry={retryBoot} />;

  // The container is keyed by auth state so navigation state never leaks between sessions, and deep links
  // (password recovery) are only resolved while signed out.
  return (
    <NavigationContainer key={status} theme={navTheme} linking={status === "anonymous" ? linking : undefined}>
      {status === "authenticated" ? (
        <AuthenticatedApp />
      ) : (
        <Stack.Navigator screenOptions={{ headerShown: false }}>
          <Stack.Screen name="Auth" component={AuthScreen} />
          <Stack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
          <Stack.Screen name="ResetPassword" component={ResetPasswordScreen} />
        </Stack.Navigator>
      )}
    </NavigationContainer>
  );
}
