import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidPassword } from '../../lib/validation';
import type { AuthScreenProps } from '../../navigation/types';
import { Button, Input, useToast } from '../../ui';
import { AuthLayout } from './AuthLayout';

export function ResetPasswordScreen({ navigation }: AuthScreenProps<'ResetPassword'>) {
  const toast = useToast();
  const onError = useErrorToast();
  const [token, setToken] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const reset = useMutation({
    mutationFn: () => authApi.resetPassword({ token: token.trim(), password }),
    onSuccess: () => {
      toast.success(fr.auth.resetDone);
      navigation.popToTop();
      navigation.navigate('Login');
    },
    onError,
  });

  const tokenError = submitted && token.trim().length === 0 ? fr.validation.tokenRequired : null;
  const passwordError =
    submitted && !isValidPassword(password) ? fr.validation.passwordLength : null;
  const confirmError = submitted && confirm !== password ? fr.validation.passwordMismatch : null;

  const submit = () => {
    setSubmitted(true);
    if (token.trim().length === 0 || !isValidPassword(password) || confirm !== password) return;
    reset.mutate();
  };

  return (
    <AuthLayout title={fr.auth.resetTitle} subtitle={fr.auth.resetSubtitle}>
      <Input
        label={fr.auth.resetToken}
        value={token}
        onChangeText={setToken}
        error={tokenError}
        autoCapitalize="none"
        autoCorrect={false}
      />
      <Input
        label={fr.auth.newPassword}
        helper={fr.auth.passwordHelper}
        value={password}
        onChangeText={setPassword}
        error={passwordError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="new-password"
        maxLength={72}
      />
      <Input
        label={fr.auth.confirmPassword}
        value={confirm}
        onChangeText={setConfirm}
        error={confirmError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="new-password"
        maxLength={72}
      />
      <Button label={fr.auth.resetCta} onPress={submit} loading={reset.isPending} />
    </AuthLayout>
  );
}
