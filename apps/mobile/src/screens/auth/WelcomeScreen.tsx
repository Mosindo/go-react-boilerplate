import React from 'react';
import { StyleSheet, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { fr } from '../../i18n/fr';
import type { AuthStackParamList } from '../../navigation/types';
import { Button, Text } from '../../shared/ui';
import { spacing } from '../../theme';
import { Brand } from './Brand';

export function WelcomeScreen({
  navigation,
}: NativeStackScreenProps<AuthStackParamList, 'Welcome'>) {
  const insets = useSafeAreaInsets();
  return (
    <View
      style={[
        styles.root,
        { paddingTop: insets.top + spacing.xxl, paddingBottom: insets.bottom + spacing.xl },
      ]}
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <Brand light />
        <Text variant="title" style={styles.title}>
          {fr.auth.welcomeTitle}
        </Text>
        <Text style={styles.body}>{fr.auth.welcomeBody}</Text>
      </View>
      <Button
        label={fr.auth.createAccount}
        onPress={() => navigation.navigate('Register')}
        testID="welcome-register"
      />
      <Button
        label={fr.auth.login}
        variant="secondary"
        onPress={() => navigation.navigate('Login')}
        style={styles.second}
        testID="welcome-login"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#2A1633',
    paddingHorizontal: spacing.xl,
    justifyContent: 'flex-end',
  },
  hero: { flex: 1, justifyContent: 'center' },
  title: { color: '#FBF6F0', marginBottom: spacing.md },
  body: { color: '#D9C7E0' },
  second: { marginTop: spacing.md },
});
