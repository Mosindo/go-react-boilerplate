import { Ionicons } from "@expo/vector-icons";
import { useFonts } from "expo-font";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import React, { useState } from "react";
import { SafeAreaProvider } from "react-native-safe-area-context";

import { ApiError } from "./api/client";
import { AuthProvider } from "./auth/AuthContext";
import { RootNavigator } from "./navigation/RootNavigator";
import { RealtimeProvider } from "./realtime/RealtimeProvider";
import { useTheme } from "./theme";
import { FeedbackProvider } from "./ui/Feedback";
import { Loading } from "./ui/States";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 15_000,
        // Retrying a 4xx never helps; only transient failures are retried.
        retry: (count, error) =>
          !(error instanceof ApiError && error.status >= 400 && error.status < 500) && count < 2,
      },
    },
  });
}

function Themed() {
  const { isDark } = useTheme();
  // Icon fonts load asynchronously on web; wait so icons never pop in after first paint.
  const [fontsLoaded] = useFonts(Ionicons.font);
  if (!fontsLoaded) return <Loading />;
  return (
    <>
      <StatusBar style={isDark ? "light" : "dark"} />
      <RootNavigator />
    </>
  );
}

export default function App() {
  const [client] = useState(createQueryClient);
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={client}>
        <FeedbackProvider>
          <AuthProvider>
            <RealtimeProvider>
              <Themed />
            </RealtimeProvider>
          </AuthProvider>
        </FeedbackProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
