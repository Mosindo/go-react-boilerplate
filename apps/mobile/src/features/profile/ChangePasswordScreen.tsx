import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { passwordProblem } from "../../lib/validation";
import type { AppStackParamList } from "../../navigation/types";
import { Button, Header, Screen, showToast, Text, TextField } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "ChangePassword">;

export function ChangePasswordScreen({ navigation }: Props) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      showToast("Mot de passe modifié. Tes autres appareils ont été déconnectés.", "success");
      navigation.goBack();
    }
  });
  const problem = submitted ? passwordProblem(next) : null;
  return (
    <Screen keyboard scroll header={<Header back title="Changer de mot de passe" />}>
      <TextField label="Mot de passe actuel" onChangeText={setCurrent} secureTextEntry textContentType="password" value={current} />
      <TextField error={problem} label="Nouveau mot de passe" onChangeText={setNext} secureTextEntry textContentType="newPassword" value={next} />
      {change.isError ? (
        <Text tone="danger">
          {change.error && "code" in change.error && change.error.code === "invalid_credentials"
            ? "Mot de passe actuel incorrect."
            : errorMessage(change.error)}
        </Text>
      ) : null}
      <Button
        disabled={!current}
        label="Modifier"
        loading={change.isPending}
        onPress={() => {
          setSubmitted(true);
          if (!passwordProblem(next)) {
            change.mutate();
          }
        }}
      />
    </Screen>
  );
}
