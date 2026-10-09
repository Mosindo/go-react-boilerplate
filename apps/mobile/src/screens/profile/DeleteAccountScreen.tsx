import React, { useState } from 'react';
import { StyleSheet, View } from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import { authApi } from '../../api/auth';
import { errorMessage, isApiError } from '../../api/errors';
import { useAuth } from '../../auth/AuthProvider';
import { fr } from '../../i18n/fr';
import { Button, ConfirmDialog, Screen, Text, TextField } from '../../shared/ui';
import { toast } from '../../shared/toastStore';
import { spacing, useTheme } from '../../theme';

export function DeleteAccountScreen() {
  const { colors } = useTheme();
  const { endSession } = useAuth();
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const ask = () => {
    if (!password) {
      setError(fr.validation.passwordRequired);
      return;
    }
    setError(null);
    setConfirming(true);
  };

  const submit = async () => {
    setBusy(true);
    try {
      await authApi.deleteAccount(password);
      setConfirming(false);
      await endSession();
      toast.show(fr.account.deleted);
    } catch (err) {
      setConfirming(false);
      if (isApiError(err) && (err.code === 'unauthorized' || err.code === 'forbidden')) {
        setError(fr.account.wrongPassword);
      } else {
        toast.error(errorMessage(err));
      }
      setBusy(false);
    }
  };

  return (
    <Screen scroll testID="delete-account-screen">
      <View style={[styles.warning, { backgroundColor: colors.dangerSoft }]}>
        <Ionicons name="warning-outline" size={22} color={colors.danger} />
        <Text style={styles.warningText}>{fr.account.deleteWarning}</Text>
      </View>
      <TextField
        testID="delete-password"
        label={fr.account.deleteConfirmPassword}
        value={password}
        onChangeText={setPassword}
        error={error}
        secure
        autoCapitalize="none"
        autoComplete="current-password"
      />
      <Button
        label={fr.account.deleteAction}
        variant="danger"
        onPress={ask}
        testID="delete-submit"
      />
      <ConfirmDialog
        visible={confirming}
        title={fr.account.deleteConfirmTitle}
        body={fr.account.deleteConfirmBody}
        confirmLabel={fr.account.deleteAction}
        destructive
        loading={busy}
        onCancel={() => setConfirming(false)}
        onConfirm={() => void submit()}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  warning: {
    flexDirection: 'row',
    padding: spacing.lg,
    borderRadius: 12,
    marginBottom: spacing.xl,
  },
  warningText: { flex: 1, marginLeft: spacing.md },
});
