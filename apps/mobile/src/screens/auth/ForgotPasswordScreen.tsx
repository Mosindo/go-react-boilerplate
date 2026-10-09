import React, { useState } from 'react';
import { StyleSheet } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { authApi } from '../../api/auth';
import { errorMessage } from '../../api/errors';
import { fr } from '../../i18n/fr';
import type { AuthStackParamList } from '../../navigation/types';
import { Button, Screen, Text, TextField } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { spacing } from '../../theme';
import { normalizeEmail, validateEmail } from '../../validators';

export function ForgotPasswordScreen({
  navigation,
}: NativeStackScreenProps<AuthStackParamList, 'ForgotPassword'>) {
  const [email, setEmail] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const problem = validateEmail(email);
    setError(problem);
    if (problem) return;
    setBusy(true);
    try {
      await authApi.forgotPassword(normalizeEmail(email));
      toast.show(fr.auth.codeSent);
      navigation.navigate('ResetPassword', { email: normalizeEmail(email) });
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Text variant="title">{fr.auth.forgotTitle}</Text>
      <Text tone="muted" style={styles.sub}>
        {fr.auth.forgotSubtitle}
      </Text>
      <TextField
        testID="forgot-email"
        label={fr.auth.email}
        value={email}
        onChangeText={setEmail}
        error={error}
        keyboardType="email-address"
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        returnKeyType="done"
        onSubmitEditing={submit}
      />
      <Button label={fr.auth.sendCode} onPress={submit} loading={busy} testID="forgot-submit" />
    </Screen>
  );
}

const styles = StyleSheet.create({ sub: { marginBottom: spacing.xl, marginTop: spacing.xs } });
