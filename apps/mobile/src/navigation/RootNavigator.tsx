import React from 'react';
import { StyleSheet, View } from 'react-native';
import { fr } from '../i18n/fr';
import { errorMessage } from '../api/errors';
import { useAuth, useMe } from '../auth/AuthProvider';
import { RealtimeProvider } from '../realtime/RealtimeProvider';
import { OnboardingScreen } from '../screens/onboarding/OnboardingScreen';
import { Button, ErrorState, LoadingView } from '../shared/ui';
import { spacing, useTheme } from '../theme';
import { AuthNavigator, MainNavigator } from './Navigators';

export function RootNavigator() {
  const { booting, signedIn, signOut } = useAuth();
  const { colors } = useTheme();
  const me = useMe(signedIn);

  if (booting) return <LoadingView label={fr.common.loading} />;
  if (!signedIn) return <AuthNavigator />;
  if (me.isPending) return <LoadingView label={fr.common.loading} />;
  if (me.isError) {
    return (
      <View style={[styles.flex, { backgroundColor: colors.background }]}>
        <ErrorState message={errorMessage(me.error)} onRetry={() => void me.refetch()} />
        <Button
          label={fr.auth.logout}
          variant="ghost"
          onPress={() => void signOut()}
          style={styles.logout}
        />
      </View>
    );
  }
  if (!me.data.onboarding.complete) return <OnboardingScreen me={me.data} />;
  return (
    <>
      <RealtimeProvider userId={me.data.id} />
      <MainNavigator />
    </>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  logout: { marginBottom: spacing.xl, marginHorizontal: spacing.xl },
});
