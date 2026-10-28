import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { changePassword } from "../../api/auth";
import { Button, Screen, Text, TextField, useToast } from "../../components";
import { errorMessage } from "../../lib/format";
import { validatePassword } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

export function ChangePasswordScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "ChangePassword">) {
  const toast = useToast();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [errors, setErrors] = useState<{ current?: string | null; next?: string | null }>({});
  const mutation = useMutation({
    mutationFn: () => changePassword(current, next),
    onSuccess: () => {
      toast.show("Password changed", "success");
      navigation.goBack();
    }
  });

  const submit = (): void => {
    const e = { current: current ? null : "Enter your current password.", next: validatePassword(next) };
    setErrors(e);
    if (!e.current && !e.next) mutation.mutate();
  };

  return (
    <Screen scroll>
      <View style={{ gap: spacing.lg }}>
        <TextField label="Current password" value={current} onChangeText={setCurrent} secureTextEntry error={errors.current} />
        <TextField label="New password" value={next} onChangeText={setNext} secureTextEntry error={errors.next} hint="At least 8 characters." />
        {mutation.isError ? <Text tone="danger">{errorMessage(mutation.error)}</Text> : null}
        <Button title="Change password" onPress={submit} loading={mutation.isPending} />
      </View>
    </Screen>
  );
}
