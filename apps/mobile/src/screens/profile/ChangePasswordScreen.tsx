import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { changePassword } from "../../api/auth";
import type { AppStackParamList } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Input } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

type Props = NativeStackScreenProps<AppStackParamList, "ChangePassword">;

export default function ChangePasswordScreen({ navigation }: Props) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const mutation = useMutation({
    mutationFn: () => changePassword(current, next),
    onSuccess: () => {
      showToast("Mot de passe modifié. Tes autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    }
  });
  return (
    <Screen>
      <ScreenHeader title="Changer de mot de passe" />
      <Input autoComplete="current-password" label="Mot de passe actuel" onChangeText={setCurrent} secureTextEntry value={current} />
      <Input
        autoComplete="new-password"
        error={mutation.isError ? errorMessage(mutation.error) : null}
        helperText="8 caractères minimum, avec une lettre et un chiffre."
        label="Nouveau mot de passe"
        onChangeText={setNext}
        secureTextEntry
        value={next}
      />
      <Button disabled={!current || next.length < 8} label="Enregistrer" loading={mutation.isPending} onPress={() => mutation.mutate()} />
    </Screen>
  );
}
