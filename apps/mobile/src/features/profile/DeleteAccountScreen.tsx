import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { authApi } from '../../api';
import { fr } from '../../i18n';
import { useErrorToast } from '../../lib/useErrorToast';
import { useStyles } from '../../theme';
import { Button, Card, ConfirmDialog, Input, Screen, Text, useToast } from '../../ui';
import { useAuth } from '../auth/AuthContext';

export function DeleteAccountScreen() {
  const toast = useToast();
  const onError = useErrorToast();
  const { endSession } = useAuth();
  const [password, setPassword] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const styles = useStyles((t) => ({
    warning: {
      backgroundColor: t.colors.dangerSoft,
      borderColor: t.colors.danger,
      gap: t.spacing.sm,
    },
  }));

  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      setConfirming(false);
      await endSession();
      toast.success(fr.deleteAccount.done);
    },
    onError: (error) => {
      setConfirming(false);
      onError(error);
    },
  });

  const error = submitted && password.length === 0 ? fr.validation.passwordRequired : null;
  const submit = () => {
    setSubmitted(true);
    if (password.length === 0) return;
    setConfirming(true);
  };

  return (
    <Screen scroll edges={['bottom']}>
      <Card style={styles.warning}>
        <Text variant="heading" color="danger">
          {fr.deleteAccount.warningTitle}
        </Text>
        <Text>{fr.deleteAccount.warning}</Text>
      </Card>
      <Input
        label={fr.deleteAccount.passwordLabel}
        value={password}
        onChangeText={setPassword}
        error={error}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="current-password"
        testID="delete-password"
      />
      <Button label={fr.deleteAccount.cta} variant="danger" onPress={submit} />
      <ConfirmDialog
        visible={confirming}
        title={fr.deleteAccount.confirmTitle}
        message={fr.deleteAccount.confirmMessage}
        confirmLabel={fr.deleteAccount.confirm}
        destructive
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
        onCancel={() => setConfirming(false)}
      />
    </Screen>
  );
}
