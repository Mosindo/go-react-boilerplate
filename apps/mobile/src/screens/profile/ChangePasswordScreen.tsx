import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { changePassword } from "../../api/auth";
import { validatePassword } from "../../lib/validation";
import type { RootStackParamList } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, FormField, Input, Notice, Text } from "../../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "ChangePassword">;

export function ChangePasswordScreen({ navigation }: Props) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const problem = validatePassword(next);
    if (!current) return setError("Saisissez votre mot de passe actuel.");
    if (problem) return setError(problem);
    setBusy(true);
    setError(null);
    try {
      await changePassword(current, next);
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Modification impossible.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="change-password-screen">
      <Text tone="muted">Pour votre sécurité, vos autres appareils seront déconnectés.</Text>
      {error ? <Notice message={error} /> : null}
      <FormField label="Mot de passe actuel">
        <Input
          testID="cp-current"
          value={current}
          onChangeText={setCurrent}
          secureTextEntry
          autoCapitalize="none"
          autoComplete="current-password"
        />
      </FormField>
      <FormField label="Nouveau mot de passe" hint="8 caractères minimum.">
        <Input testID="cp-new" value={next} onChangeText={setNext} secureTextEntry autoCapitalize="none" autoComplete="new-password" />
      </FormField>
      <Button testID="cp-submit" label="Modifier" loading={busy} onPress={() => void submit()} />
    </ScreenContainer>
  );
}
