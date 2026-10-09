import React, { useEffect } from 'react';
import { AppState, Platform, StyleSheet, View } from 'react-native';
import { StatusBar } from 'expo-status-bar';
import {
  DarkTheme,
  DefaultTheme,
  NavigationContainer,
  type Theme as NavTheme,
} from '@react-navigation/native';
import { focusManager, QueryClientProvider } from '@tanstack/react-query';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { AuthProvider } from './src/auth/AuthProvider';
import { fr } from './src/i18n/fr';
import { navigationRef } from './src/navigation/ref';
import { RootNavigator } from './src/navigation/RootNavigator';
import { queryClient } from './src/query/client';
import { realtimeStatus } from './src/realtime/RealtimeProvider';
import { MatchModal } from './src/screens/MatchModal';
import { Text, ToastHost } from './src/shared/ui';
import { MAX_CONTENT_WIDTH, ThemeProvider, spacing, useTheme } from './src/theme';

function useReactQueryAppFocus() {
  useEffect(() => {
    if (Platform.OS === 'web') return;
    const sub = AppState.addEventListener('change', (s) => focusManager.setFocused(s === 'active'));
    return () => sub.remove();
  }, []);
}

function ConnectionBanner() {
  const status = realtimeStatus.use();
  const { colors } = useTheme();
  if (status !== 'reconnecting') return null;
  return (
    <View
      style={[styles.banner, { backgroundColor: colors.accent }]}
      accessibilityLiveRegion="polite"
      testID="reconnecting-banner"
    >
      <Text variant="caption" style={{ color: colors.onAccent }}>
        {fr.realtime.reconnecting}
      </Text>
    </View>
  );
}

function Shell() {
  const { colors, dark } = useTheme();
  useReactQueryAppFocus();
  const navTheme: NavTheme = {
    ...(dark ? DarkTheme : DefaultTheme),
    colors: {
      ...(dark ? DarkTheme : DefaultTheme).colors,
      primary: colors.primary,
      background: colors.background,
      card: colors.surface,
      text: colors.text,
      border: colors.border,
      notification: colors.primary,
    },
  };
  return (
    // On wide screens (web) the app lives in a centred phone-width column.
    <View style={[styles.outside, { backgroundColor: colors.outside }]}>
      <View style={[styles.column, { backgroundColor: colors.background }]}>
        <StatusBar style={dark ? 'light' : 'dark'} />
        <NavigationContainer
          ref={navigationRef}
          theme={navTheme}
          documentTitle={{ formatter: () => fr.app.name }}
        >
          <RootNavigator />
        </NavigationContainer>
        <ConnectionBanner />
        <MatchModal />
        <ToastHost />
      </View>
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <Shell />
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  outside: { flex: 1, alignItems: 'center' },
  column: { flex: 1, width: '100%', maxWidth: MAX_CONTENT_WIDTH, overflow: 'hidden' },
  banner: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: 'center',
    padding: spacing.xs,
  },
});
