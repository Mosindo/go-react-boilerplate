import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { isValidPassword } from '../../lib/validation';
import type { RootScreenProps } from '../../navigation/types';
import { Button, Input, Screen, useToast } from '../../ui';

export function ChangePasswordScreen({ navigation }: RootScreenProps<'ChangePassword'>) {
  const toast = useToast();
  const onError = useErrorToast();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [submitted, setSubmitted] = useState(false);

  const change = useMutation({
    mutationFn: () => authApi.changePassword({ currentPassword: current, newPassword: next }),
    onSuccess: () => {
      toast.success(fr.auth.passwordChanged);
      navigation.goBack();
    },
    onError,
  });

  const currentError = submitted && current.length === 0 ? fr.validation.passwordRequired : null;
  const nextError = submitted && !isValidPassword(next) ? fr.validation.passwordLength : null;
  const confirmError = submitted && confirm !== next ? fr.validation.passwordMismatch : null;

  const submit = () => {
    setSubmitted(true);
    if (current.length === 0 || !isValidPassword(next) || confirm !== next) return;
    change.mutate();
  };

  return (
    <Screen scroll edges={['bottom']}>
      <Input
        label={fr.auth.currentPassword}
        value={current}
        onChangeText={setCurrent}
        error={currentError}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="current-password"
      />
      <Input
        label={fr.auth.newPassword}
        helper={fr.auth.passwordHelper}
        value={next}
        onChangeText={setNext}
        error={nextError}
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
      <Button label={fr.auth.changePasswordCta} onPress={submit} loading={change.isPending} />
    </Screen>
  );
}
