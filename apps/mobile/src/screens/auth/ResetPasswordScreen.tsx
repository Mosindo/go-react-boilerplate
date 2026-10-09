import React, { useRef, useState } from 'react';
import { StyleSheet, TextInput } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { authApi } from '../../api/auth';
import { errorMessage } from '../../api/errors';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import type { AuthStackParamList } from '../../navigation/types';
import { Button, Screen, Text, TextField } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { spacing } from '../../theme';
import { validatePassword, validateResetCode } from '../../validators';

export function ResetPasswordScreen({
  route,
  navigation,
}: NativeStackScreenProps<AuthStackParamList, 'ResetPassword'>) {
  const { email } = route.params;
  const { endSession } = useAuth();
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<{ code?: string; password?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);
  const [resending, setResending] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    const next = {
      code: validateResetCode(code) ?? undefined,
      password: validatePassword(password) ?? undefined,
    };
    setErrors(next);
    if (next.code || next.password) return;
    setBusy(true);
    try {
      await authApi.resetPassword(email, code.trim(), password);
      await endSession();
      toast.success(fr.auth.resetDone);
      navigation.popTo('Login');
    } catch (err) {
      setErrors({ form: errorMessage(err) });
    } finally {
      setBusy(false);
    }
  };

  const resend = async () => {
    setResending(true);
    try {
      await authApi.forgotPassword(email);
      toast.show(fr.auth.codeSent);
    } catch (err) {
      toast.error(errorMessage(err));
    } finally {
      setResending(false);
    }
  };

  return (
    <Screen scroll testID="reset-screen">
      <Text variant="title">{fr.auth.resetTitle}</Text>
      <Text tone="muted" style={styles.sub}>
        {fr.auth.resetSubtitle(email)}
      </Text>
      <TextField
        testID="reset-code"
        label={fr.auth.code}
        value={code}
        onChangeText={setCode}
        error={errors.code}
        autoCapitalize="none"
        autoCorrect={false}
        maxLength={8}
        returnKeyType="next"
        onSubmitEditing={() => passwordRef.current?.focus()}
      />
      <TextField
        ref={passwordRef}
        testID="reset-password"
        label={fr.auth.newPassword}
        hint={fr.auth.passwordHint}
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secure
        autoCapitalize="none"
        autoComplete="new-password"
        returnKeyType="done"
        onSubmitEditing={submit}
      />
      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert" testID="reset-error" style={styles.formError}>
          {errors.form}
        </Text>
      ) : null}
      <Button label={fr.auth.resetAction} onPress={submit} loading={busy} testID="reset-submit" />
      <Button
        label={fr.auth.resendCode}
        variant="ghost"
        onPress={resend}
        loading={resending}
        style={styles.link}
        testID="reset-resend"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  sub: { marginBottom: spacing.xl, marginTop: spacing.xs },
  formError: { marginBottom: spacing.md },
  link: { marginTop: spacing.sm },
});
