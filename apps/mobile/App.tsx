import React, { useState } from "react";
import { StatusBar } from "expo-status-bar";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryClient } from "./src/api/queryClient";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import { RealtimeProvider } from "./src/hooks/useRealtime";
import { RootNavigator } from "./src/navigation/RootNavigator";
import AuthScreen from "./src/screens/AuthScreen";
import OnboardingScreen from "./src/screens/OnboardingScreen";
import { LoadingView, Toast } from "./src/shared/feedback";
import { SafeAreaLayout } from "./src/shared/layout";
import { ThemeProvider, useTheme } from "./src/shared/ui";

function Shell() {
  const { isBooting, isAuthenticated, user } = useAuth();
  const { scheme } = useTheme();

  let body: React.ReactNode;
  if (isBooting) {
    body = (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement…" />
      </SafeAreaLayout>
    );
  } else if (!isAuthenticated || !user) {
    body = <AuthScreen />;
  } else if (!user.profileComplete) {
    body = <OnboardingScreen key={user.id} />;
  } else {
    body = <RootNavigator key={user.id} />;
  }

  return (
    <>
      <StatusBar style={scheme === "dark" ? "light" : "dark"} />
      {body}
      <Toast />
    </>
  );
}

export default function App() {
  const [queryClient] = useState(createQueryClient);
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <RealtimeProvider>
            <Shell />
          </RealtimeProvider>
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
