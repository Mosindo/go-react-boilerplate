import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../api/client";
import type { AuthScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Header, Screen, Text, TextField, showToast } from "../../ui";
import { validatePassword } from "../../utils/validation";

export function ResetPasswordScreen({ navigation, route }: AuthScreenProps<"ResetPassword">) {
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(code, password),
    onSuccess: () => {
      showToast("Mot de passe modifié. Vous pouvez vous connecter.", "success");
      navigation.navigate("Login");
    }
  });

  const codeError = code.replace(/[\s-]/g, "").length !== 10 ? "Le code contient 10 caractères." : null;
  const passwordError = validatePassword(password);
  const submit = () => {
    setSubmitted(true);
    if (!codeError && !passwordError) {
      reset.mutate();
    }
  };

  return (
    <Screen scroll testID="reset-screen">
      <Header onBack={navigation.goBack} />
      <View style={styles.form}>
        <Text variant="title">Nouveau mot de passe</Text>
        <Text variant="small" color="textMuted">
          {route.params?.email
            ? `Si un compte existe pour ${route.params.email}, un code vient de lui être envoyé.`
            : "Saisissez le code reçu par e-mail."}
        </Text>
        <TextField
          label="Code reçu"
          value={code}
          onChangeText={(v) => setCode(v.toUpperCase())}
          autoCapitalize="characters"
          autoCorrect={false}
          placeholder="ABCDE-FGHJK"
          error={submitted ? codeError : null}
        />
        <TextField
          label="Nouveau mot de passe"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          autoComplete="new-password"
          hint="8 caractères minimum, avec une lettre et un chiffre."
          error={submitted ? passwordError : null}
          onSubmitEditing={submit}
        />
        {reset.error ? (
          <Text variant="small" color="danger">
            {errorMessage(reset.error)}
          </Text>
        ) : null}
        <Button label="Réinitialiser" onPress={submit} loading={reset.isPending} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: {
    gap: spacing.md,
    paddingTop: spacing.md
  }
});
