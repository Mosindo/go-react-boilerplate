import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import { passwordProblem } from "../../lib/validation";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField, toast } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "ChangePassword">;

export function ChangePasswordScreen({ navigation }: Props) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    const problem = passwordProblem(next);
    if (problem) {
      setError(problem);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await authApi.changePassword(current, next);
      toast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Screen scroll edges={["top", "bottom"]}>
      <Header title="Changer de mot de passe" onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        <TextField label="Mot de passe actuel" value={current} onChangeText={setCurrent} secureToggle autoComplete="current-password" />
        <TextField
          label="Nouveau mot de passe"
          value={next}
          onChangeText={setNext}
          secureToggle
          autoComplete="new-password"
          hint="8 caractères minimum, avec une lettre et un chiffre."
        />
        <InlineError message={error} />
        <Button label="Enregistrer" onPress={submit} loading={loading} disabled={!current || !next} />
        <Text variant="caption" color="textSubtle">
          Par sécurité, toutes vos autres sessions seront fermées.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({ form: { gap: spacing.md } });
