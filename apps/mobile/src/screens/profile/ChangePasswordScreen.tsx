import React, { useRef, useState } from 'react';
import { TextInput } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { authApi } from '../../api/auth';
import { errorMessage, isApiError } from '../../api/errors';
import { fr } from '../../i18n/fr';
import type { MainStackParamList } from '../../navigation/types';
import { Button, Screen, TextField } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { validatePassword } from '../../validators';

export function ChangePasswordScreen({
  navigation,
}: NativeStackScreenProps<MainStackParamList, 'ChangePassword'>) {
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [errors, setErrors] = useState<{ current?: string; next?: string; confirm?: string }>({});
  const [busy, setBusy] = useState(false);
  const nextRef = useRef<TextInput>(null);
  const confirmRef = useRef<TextInput>(null);

  const submit = async () => {
    const e = {
      current: current ? undefined : fr.validation.passwordRequired,
      next: validatePassword(next) ?? undefined,
      confirm: next === confirm ? undefined : fr.validation.passwordMismatch,
    };
    setErrors(e);
    if (e.current || e.next || e.confirm) return;
    setBusy(true);
    try {
      await authApi.changePassword(current, next);
      toast.success(fr.account.passwordChanged);
      navigation.goBack();
    } catch (err) {
      if (isApiError(err) && (err.code === 'unauthorized' || err.code === 'forbidden')) {
        setErrors({ current: fr.account.wrongPassword });
      } else {
        toast.error(errorMessage(err));
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="change-password-screen">
      <TextField
        testID="cp-current"
        label={fr.account.currentPassword}
        value={current}
        onChangeText={setCurrent}
        error={errors.current}
        secure
        autoCapitalize="none"
        autoComplete="current-password"
        returnKeyType="next"
        onSubmitEditing={() => nextRef.current?.focus()}
      />
      <TextField
        ref={nextRef}
        testID="cp-new"
        label={fr.auth.newPassword}
        hint={fr.auth.passwordHint}
        value={next}
        onChangeText={setNext}
        error={errors.next}
        secure
        autoCapitalize="none"
        autoComplete="new-password"
        returnKeyType="next"
        onSubmitEditing={() => confirmRef.current?.focus()}
      />
      <TextField
        ref={confirmRef}
        testID="cp-confirm"
        label={fr.auth.confirmPassword}
        value={confirm}
        onChangeText={setConfirm}
        error={errors.confirm}
        secure
        autoCapitalize="none"
        returnKeyType="done"
        onSubmitEditing={submit}
      />
      <Button label={fr.common.save} onPress={submit} loading={busy} testID="cp-submit" />
    </Screen>
  );
}
