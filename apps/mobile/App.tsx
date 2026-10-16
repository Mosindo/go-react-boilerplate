import React, { useEffect } from "react";
import { StatusBar } from "expo-status-bar";
import { StyleSheet, View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import AuthScreen from "./src/screens/AuthScreen";
import OnboardingScreen from "./src/screens/OnboardingScreen";
import { useMyProfile } from "./src/hooks/queries";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import { RealtimeProvider } from "./src/hooks/useRealtime";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { clearGlobalError, ErrorView, LoadingView, useGlobalFeedback } from "./src/shared/feedback";
import { SafeAreaLayout } from "./src/shared/layout";
import { Button, colors, spacing } from "./src/shared/ui";

function SignedIn() {
  const me = useMyProfile();
  const { signOut } = useAuth();

  if (me.isLoading) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement de votre profil…" />
      </SafeAreaLayout>
    );
  }
  if (!me.data) {
    return (
      <SafeAreaLayout style={styles.padded}>
        <ErrorView message="Impossible de charger votre profil. Vérifiez votre connexion." onAction={() => void me.refetch()} />
        <Button label="Se déconnecter" onPress={() => void signOut()} variant="ghost" />
      </SafeAreaLayout>
    );
  }
  if (!me.data.isComplete) {
    return <OnboardingScreen me={me.data} />;
  }
  return (
    <RealtimeProvider>
      <RootNavigator />
    </RealtimeProvider>
  );
}

function AppShell() {
  const { status } = useAuth();
  const feedback = useGlobalFeedback();

  useEffect(() => {
    if (!feedback.error) {
      return;
    }
    const timeoutId = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(timeoutId);
  }, [feedback.error]);

  let content: React.ReactNode;
  if (status === "booting") {
    content = (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Ouverture de votre session…" />
      </SafeAreaLayout>
    );
  } else if (status === "signedOut") {
    content = <AuthScreen />;
  } else {
    content = <SignedIn />;
  }

  return (
    <>
      <StatusBar style="dark" />
      {content}
      {feedback.error ? (
        <View pointerEvents="box-none" style={styles.bannerWrap}>
          <ErrorView actionLabel="Fermer" compact message={feedback.error} onAction={clearGlobalError} title="Problème de connexion" />
        </View>
      ) : null}
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  padded: { padding: spacing.lg, gap: spacing.md, backgroundColor: colors.background },
  bannerWrap: { position: "absolute", left: spacing.lg, right: spacing.lg, bottom: spacing.xxl }
});
