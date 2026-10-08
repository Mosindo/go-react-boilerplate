import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { changePassword } from "../api/auth";
import { describeError } from "../api/client";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, FormField, Notice, Text } from "../shared/ui";
import { validatePassword } from "../utils/validation";

export default function ChangePasswordScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "ChangePassword">) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setTouched(true);
    setError(null);
    if (!current || validatePassword(next)) return;
    setBusy(true);
    try {
      await changePassword(current, next);
      showToast("Mot de passe modifié. Vos autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    } catch (e) {
      setError(describeError(e));
      setBusy(false);
    }
  };

  return (
    <ScreenContainer>
      <Text tone="muted">Vos autres appareils seront déconnectés après le changement.</Text>
      {error ? <Notice message={error} tone="error" /> : null}
      <FormField
        autoCapitalize="none"
        error={touched && !current ? "Saisissez votre mot de passe actuel." : null}
        label="Mot de passe actuel"
        onChangeText={setCurrent}
        secureTextEntry
        value={current}
      />
      <FormField
        autoCapitalize="none"
        error={touched ? validatePassword(next) : null}
        hint="8 caractères minimum, avec des lettres et des chiffres ou symboles."
        label="Nouveau mot de passe"
        onChangeText={setNext}
        secureTextEntry
        value={next}
      />
      <Button fullWidth loading={busy} onPress={submit} size="lg">
        Modifier le mot de passe
      </Button>
    </ScreenContainer>
  );
}
