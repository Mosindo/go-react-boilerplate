import { DarkTheme, DefaultTheme, NavigationContainer } from '@react-navigation/native';
import type { Theme as NavTheme } from '@react-navigation/native';
import { useMemo } from 'react';
import { useAuth } from '../features/auth/AuthContext';
import { fr } from '../i18n';
import { useTheme } from '../theme';
import { ErrorState, Loading, Screen } from '../ui';
import { AuthNavigator } from './AuthNavigator';
import { MainNavigator } from './MainNavigator';
import { OnboardingNavigator } from './OnboardingNavigator';
import { navigationRef } from './ref';

export function RootNavigator() {
  const { colors, scheme } = useTheme();
  const { status, user, retryBoot } = useAuth();

  const navTheme = useMemo<NavTheme>(() => {
    const base = scheme === 'dark' ? DarkTheme : DefaultTheme;
    return {
      ...base,
      colors: {
        ...base.colors,
        primary: colors.primary,
        background: colors.bg,
        card: colors.bg,
        text: colors.text,
        border: colors.border,
        notification: colors.primary,
      },
    };
  }, [colors, scheme]);

  let content;
  if (status === 'booting') {
    content = <Loading />;
  } else if (status === 'bootError') {
    content = (
      <Screen edges={['top', 'bottom']}>
        <ErrorState
          message={fr.errors.network}
          onRetry={retryBoot}
          title={fr.errors.networkTitle}
        />
      </Screen>
    );
  } else if (status === 'signedOut' || !user) {
    content = <AuthNavigator />;
  } else if (!user.profileComplete) {
    content = <OnboardingNavigator />;
  } else {
    content = <MainNavigator />;
  }

  return (
    <NavigationContainer
      ref={navigationRef}
      theme={navTheme}
      documentTitle={{ formatter: () => fr.brand.name }}
    >
      {content}
    </NavigationContainer>
  );
}
