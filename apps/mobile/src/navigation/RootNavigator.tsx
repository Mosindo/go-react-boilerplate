import { NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React from "react";
import { StyleSheet, View } from "react-native";
import { needsOnboarding, useAuth } from "../hooks/useAuth";
import { strings } from "../i18n/strings";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import AuthScreen from "../screens/AuthScreen";
import ChatScreen from "../screens/ChatScreen";
import OnboardingScreen from "../screens/OnboardingScreen";
import UserProfileScreen from "../screens/UserProfileScreen";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { spacing } from "../shared/ui";
import TabNavigator from "./TabNavigator";
import { useNavigationTheme, useStackScreenOptions } from "./theme";
import type { RootStackParamList } from "./types";

const Stack = createNativeStackNavigator<RootStackParamList>();

export default function RootNavigator() {
  const { bootError, isAuthenticated, isBooting, retryBoot, user } = useAuth();
  const navigationTheme = useNavigationTheme();
  const screenOptions = useStackScreenOptions();

  if (bootError) {
    return (
      <SafeAreaLayout style={styles.center}>
        <ErrorView message={bootError} onAction={retryBoot} title={strings.auth.bootErrorTitle} />
      </SafeAreaLayout>
    );
  }
  if (isBooting) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label={strings.auth.restoring} />
      </SafeAreaLayout>
    );
  }

  if (!isAuthenticated) {
    return (
      <NavigationContainer theme={navigationTheme}>
        <Stack.Navigator
          screenOptions={{ ...screenOptions, headerShown: false, animationTypeForReplace: "pop" }}
        >
          <Stack.Screen component={AuthScreen} name="Auth" />
        </Stack.Navigator>
      </NavigationContainer>
    );
  }

  if (needsOnboarding(user)) {
    return (
      <NavigationContainer theme={navigationTheme}>
        <Stack.Navigator screenOptions={{ ...screenOptions, headerShown: false }}>
          <Stack.Screen component={OnboardingScreen} name="Onboarding" />
        </Stack.Navigator>
      </NavigationContainer>
    );
  }

  // Signed in with a complete profile. The key resets navigation state when the account changes.
  return (
    <RealtimeProvider key={user?.id}>
      <View style={styles.flex}>
        <NavigationContainer theme={navigationTheme}>
          <Stack.Navigator screenOptions={screenOptions}>
            <Stack.Screen component={TabNavigator} name="Tabs" options={{ headerShown: false }} />
            <Stack.Screen
              component={ChatScreen}
              name="Chat"
              options={({ route }) => ({ title: route.params.user.firstName })}
            />
            <Stack.Screen
              component={UserProfileScreen}
              name="UserProfile"
              options={{ title: strings.tabs.profile }}
            />
          </Stack.Navigator>
        </NavigationContainer>
      </View>
    </RealtimeProvider>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  center: { justifyContent: "center", padding: spacing.lg }
});
