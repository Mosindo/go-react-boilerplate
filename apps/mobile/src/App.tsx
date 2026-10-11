import { useState } from 'react';
import { QueryClientProvider } from '@tanstack/react-query';
import { StatusBar } from 'expo-status-bar';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { AuthProvider } from './features/auth/AuthContext';
import { createQueryClient } from './lib/queryClient';
import { RootNavigator } from './navigation/RootNavigator';
import { ThemeProvider, useTheme } from './theme';
import { AppFrame, ToastProvider } from './ui';

function ThemedStatusBar() {
  const { scheme } = useTheme();
  return <StatusBar style={scheme === 'dark' ? 'light' : 'dark'} />;
}

export default function App() {
  const [queryClient] = useState(createQueryClient);
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <ThemedStatusBar />
        <AppFrame>
          <QueryClientProvider client={queryClient}>
            <ToastProvider>
              <AuthProvider>
                <RootNavigator />
              </AuthProvider>
            </ToastProvider>
          </QueryClientProvider>
        </AppFrame>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
