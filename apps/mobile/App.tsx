import React from "react";
import { StatusBar } from "expo-status-bar";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./src/auth/AuthContext";
import { MatchProvider } from "./src/features/match/MatchContext";
import { RootNavigator } from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeContext";

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 15_000, refetchOnWindowFocus: false } }
});

/** The realtime socket only exists while a user is signed in. */
function SignedInServices({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  return user ? <RealtimeProvider key={user.id}>{children}</RealtimeProvider> : <>{children}</>;
}

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <SignedInServices>
            <MatchProvider>
              <StatusBar style="auto" />
              <RootNavigator />
            </MatchProvider>
          </SignedInServices>
        </AuthProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
