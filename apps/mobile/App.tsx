import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider } from "./src/auth/AuthContext";
import { ToastProvider } from "./src/components/Toast";
import RootNavigator from "./src/navigation/RootNavigator";
import { RealtimeProvider } from "./src/realtime/RealtimeContext";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
    mutations: { retry: 0 }
  }
});

export default function App() {
  return (
    <SafeAreaProvider>
      <QueryClientProvider client={queryClient}>
        <ToastProvider>
          <AuthProvider>
            <RealtimeProvider>
              <RootNavigator />
            </RealtimeProvider>
          </AuthProvider>
        </ToastProvider>
      </QueryClientProvider>
    </SafeAreaProvider>
  );
}
