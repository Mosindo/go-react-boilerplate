import React, { useState } from "react";
import { Alert } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { authApi, errorMessage } from "../../api";
import { showToast } from "../../shared/feedback";
import { Button, Screen, Text, TextField } from "../../shared/ui";
import { validatePassword, validatePasswordConfirmation } from "../../shared/validation";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../auth/AuthProvider";

export function ChangePasswordScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "ChangePassword">) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [errors, setErrors] = useState<{ current?: string; next?: string; confirmation?: string }>({});

  const change = useMutation({
    mutationFn: authApi.changePassword,
    meta: { silent: true },
    onSuccess: () => {
      showToast("Mot de passe modifié.");
      navigation.goBack();
    }
  });

  const submit = () => {
    const nextErrors = {
      current: current ? undefined : "Saisissez votre mot de passe actuel.",
      next: validatePassword(next) ?? undefined,
      confirmation: validatePasswordConfirmation(next, confirmation) ?? undefined
    };
    setErrors(nextErrors);
    if (!nextErrors.current && !nextErrors.next && !nextErrors.confirmation) {
      change.mutate({ currentPassword: current, newPassword: next });
    }
  };

  return (
    <Screen
      footer={
        <Button label="Modifier le mot de passe" loading={change.isPending} onPress={submit} testID="password-submit" />
      }
      scroll
      testID="change-password-screen"
    >
      <TextField
        autoCapitalize="none"
        error={errors.current}
        label="Mot de passe actuel"
        onChangeText={setCurrent}
        secureTextEntry
        testID="password-current"
        value={current}
      />
      <TextField
        autoCapitalize="none"
        error={errors.next}
        label="Nouveau mot de passe"
        onChangeText={setNext}
        secureTextEntry
        testID="password-new"
        value={next}
      />
      <TextField
        autoCapitalize="none"
        error={errors.confirmation}
        label="Confirmer le nouveau mot de passe"
        onChangeText={setConfirmation}
        secureTextEntry
        testID="password-confirm"
        value={confirmation}
      />
      {change.isError ? (
        <Text color="danger" variant="caption">
          {errorMessage(change.error, "Modification impossible.")}
        </Text>
      ) : null}
    </Screen>
  );
}

export function DeleteAccountScreen() {
  const { endSession } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const remove = useMutation({
    mutationFn: authApi.deleteAccount,
    meta: { silent: true },
    onSuccess: () => endSession()
  });

  const submit = () => {
    if (!password) {
      setError("Saisissez votre mot de passe pour confirmer.");
      return;
    }
    setError(null);
    Alert.alert(
      "Supprimer définitivement votre compte ?",
      "Votre profil, vos photos, vos matchs et vos messages seront effacés. Cette action est irréversible.",
      [
        { text: "Annuler", style: "cancel" },
        { text: "Supprimer", style: "destructive", onPress: () => remove.mutate(password) }
      ]
    );
  };

  return (
    <Screen
      footer={
        <Button
          label="Supprimer mon compte"
          loading={remove.isPending}
          onPress={submit}
          testID="delete-submit"
          variant="danger"
        />
      }
      scroll
      testID="delete-account-screen"
    >
      <Text color="textMuted">
        La suppression est définitive : toutes vos données sont effacées. Confirmez avec votre mot de passe.
      </Text>
      <TextField
        autoCapitalize="none"
        error={error}
        label="Mot de passe"
        onChangeText={setPassword}
        secureTextEntry
        testID="delete-password"
        value={password}
      />
      {remove.isError ? (
        <Text color="danger" variant="caption">
          {errorMessage(remove.error, "Suppression impossible.")}
        </Text>
      ) : null}
    </Screen>
  );
}
