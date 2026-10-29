import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { StyleSheet, Text } from "react-native";
import { changePassword } from "../../api/auth";
import { Button } from "../../components/Button";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { useAsyncAction } from "../../hooks/useAsyncAction";
import { validatePassword } from "../../lib/validation";
import type { RootStackParamList } from "../../navigation/types";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";

type Props = NativeStackScreenProps<RootStackParamList, "ChangePassword">;

export default function ChangePasswordScreen({ navigation }: Props) {
  const theme = useTheme();
  const toast = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [nextError, setNextError] = useState<string | null>(null);

  const submit = useAsyncAction(async () => {
    const problem = validatePassword(next);
    setNextError(problem);
    if (problem) return;
    if (!current) throw new Error("Entrez votre mot de passe actuel.");
    await changePassword(current, next);
    toast.show("Mot de passe modifié.", "success");
    navigation.goBack();
  });

  return (
    <Screen edges={["bottom"]} scroll>
      <TextField
        label="Mot de passe actuel"
        onChangeText={setCurrent}
        revealable
        secureTextEntry
        value={current}
      />
      <TextField
        error={nextError}
        hint="8 caractères minimum."
        label="Nouveau mot de passe"
        onChangeText={setNext}
        revealable
        secureTextEntry
        value={next}
      />
      {submit.error ? (
        <Text style={[styles.error, { color: theme.danger }]}>{submit.error}</Text>
      ) : null}
      <Button
        loading={submit.loading}
        onPress={() => void submit.run()}
        title="Modifier le mot de passe"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  error: { fontSize: fontSizes.sm, marginBottom: spacing.md }
});
