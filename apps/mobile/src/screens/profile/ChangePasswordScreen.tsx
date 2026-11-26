import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { changePassword } from "../../api/auth";
import { errorMessage } from "../../api/client";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { passwordProblem } from "../../utils/validation";

type Props = NativeStackScreenProps<AppStackParamList, "ChangePassword">;

export function ChangePasswordScreen({ navigation }: Props) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const save = async () => {
    const problem = passwordProblem(next);
    if (!current || problem) return setError(problem ?? "Saisissez votre mot de passe actuel.");
    setSaving(true);
    setError(null);
    try {
      await changePassword(current, next);
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen edges={["top", "bottom"]} footer={<Button label="Enregistrer" loading={saving} onPress={save} />} scroll>
      <Header onBack={navigation.goBack} title="Mot de passe" />
      <View style={styles.form}>
        <TextField autoComplete="current-password" label="Mot de passe actuel" onChangeText={setCurrent} secureToggle value={current} />
        <TextField
          autoComplete="new-password"
          hint="8 caractères minimum, dont une lettre et un chiffre."
          label="Nouveau mot de passe"
          onChangeText={setNext}
          secureToggle
          value={next}
        />
        {error ? <Text tone="danger">{error}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.lg, paddingTop: spacing.xs } });
