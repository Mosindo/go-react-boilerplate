import React from "react";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { DarkTheme, DefaultTheme, NavigationContainer } from "@react-navigation/native";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import type { AuthStackParamList } from "./src/navigation/types";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeProvider";
import AuthScreen from "./src/screens/AuthScreen";
import ForgotPasswordScreen from "./src/screens/ForgotPasswordScreen";
import OnboardingScreen from "./src/screens/OnboardingScreen";
import { ErrorView, LoadingView, ToastHost } from "./src/shared/feedback";
import { SafeAreaLayout } from "./src/shared/layout";
import { Button, useTheme } from "./src/shared/ui";
import { useOnboardingFlowActive } from "./src/state/onboardingFlow";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();

function SignedOutNavigator() {
  const { colors, scheme } = useTheme();
  const base = scheme === "dark" ? DarkTheme : DefaultTheme;
  const theme = {
    ...base,
    colors: {
      ...base.colors,
      background: colors.background,
      card: colors.surface,
      text: colors.text,
      primary: colors.primary,
      border: colors.border
    }
  };
  return (
    <NavigationContainer theme={theme}>
      <AuthStack.Navigator
        screenOptions={{
          headerShadowVisible: false,
          headerBackTitleVisible: false,
          headerStyle: { backgroundColor: colors.background },
          headerTintColor: colors.text
        }}
      >
        <AuthStack.Screen component={AuthScreen} name="Auth" options={{ headerShown: false }} />
        <AuthStack.Screen component={ForgotPasswordScreen} name="ForgotPassword" options={{ title: "Mot de passe oublié" }} />
      </AuthStack.Navigator>
    </NavigationContainer>
  );
}

function SignedInGate() {
  const { self, selfLoading, selfError, refreshSelf, signOut } = useAuth();
  const flowActive = useOnboardingFlowActive();

  if (selfLoading || (!self && !selfError)) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement de votre profil…" />
      </SafeAreaLayout>
    );
  }
  if (!self) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger votre profil. Vérifiez votre connexion." onAction={() => void refreshSelf()} />
        <Button label="Se déconnecter" onPress={() => void signOut()} style={{ marginHorizontal: 24 }} variant="ghost" />
      </SafeAreaLayout>
    );
  }
  if (!self.onboarding.complete || flowActive) {
    return <OnboardingScreen />;
  }
  return <RootNavigator />;
}

function AppShell() {
  const { status } = useAuth();
  const { scheme } = useTheme();

  let content: React.ReactNode;
  if (status === "booting") {
    content = (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Ouverture…" />
      </SafeAreaLayout>
    );
  } else if (status === "signedOut") {
    content = <SignedOutNavigator />;
  } else {
    content = <SignedInGate />;
  }

  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      {content}
      <ToastHost />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <RealtimeProvider>
          <AppShell />
        </RealtimeProvider>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
