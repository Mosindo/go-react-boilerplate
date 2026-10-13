import React, { useEffect, useState } from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import { useOwnProfile } from "./src/hooks/queries";
import { RealtimeProvider } from "./src/hooks/useRealtime";
import { queryClient } from "./src/lib/queryClient";
import { AppNavigator } from "./src/navigation/AppNavigator";
import AuthScreen from "./src/screens/AuthScreen";
import OnboardingScreen from "./src/screens/OnboardingScreen";
import { ErrorView, LoadingView, Toasts } from "./src/shared/feedback";
import { SafeAreaLayout } from "./src/shared/layout";
import { ThemeProvider, useTheme } from "./src/theme";

function SignedIn({ userId }: { userId: string }) {
  const profile = useOwnProfile();
  // The wizard stays open until the user finishes it, even though the profile
  // becomes "complete" (photo + preferences) before the last steps.
  const [wizard, setWizard] = useState<boolean | null>(null);

  useEffect(() => {
    if (wizard === null && profile.isSuccess) {
      setWizard(!profile.data?.isComplete);
    }
  }, [profile.data, profile.isSuccess, wizard]);

  if (profile.isError) {
    return (
      <SafeAreaLayout>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void profile.refetch()} />
      </SafeAreaLayout>
    );
  }
  if (wizard === null) {
    return (
      <SafeAreaLayout>
        <LoadingView />
      </SafeAreaLayout>
    );
  }
  if (wizard) {
    return <OnboardingScreen onDone={() => setWizard(false)} />;
  }
  return (
    <RealtimeProvider userId={userId}>
      <AppNavigator />
    </RealtimeProvider>
  );
}

function AppShell() {
  const { scheme } = useTheme();
  const { isAuthenticated, isBooting, user } = useAuth();

  let content: React.ReactNode;
  if (isBooting) {
    content = (
      <SafeAreaLayout>
        <LoadingView label="Chargement…" />
      </SafeAreaLayout>
    );
  } else if (!isAuthenticated || !user) {
    content = <AuthScreen />;
  } else {
    content = <SignedIn key={user.id} userId={user.id} />;
  }

  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      {content}
      <Toasts />
    </>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <AppShell />
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
