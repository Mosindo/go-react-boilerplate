import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { changePassword } from "../../api/auth";
import { errorMessage } from "../../api/client";
import { passwordProblem } from "../../lib/validation";
import type { MainScreenProps } from "../../navigation/types";
import { Button, Header, Screen, Text, TextField, showToast, spacing } from "../../ui";

export default function ChangePasswordScreen({ navigation }: MainScreenProps<"ChangePassword">) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setError(null);
    const problem = passwordProblem(next);
    if (!current || problem) {
      setError(problem ?? "Saisissez votre mot de passe actuel.");
      return;
    }
    setSaving(true);
    try {
      await changePassword(current, next);
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen footer={<Button label="Changer de mot de passe" loading={saving} onPress={submit} />} scroll>
      <Header onBack={navigation.goBack} title="Mot de passe" />
      <View style={styles.form}>
        <TextField autoComplete="current-password" label="Mot de passe actuel" onChangeText={setCurrent} secureTextEntry value={current} />
        <TextField
          autoComplete="new-password"
          hint="8 caractères minimum, avec une lettre et un chiffre."
          label="Nouveau mot de passe"
          onChangeText={setNext}
          secureTextEntry
          value={next}
        />
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md
  }
});
