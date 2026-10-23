import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { resetPassword } from "../../api";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { validatePassword, validateRequired } from "../../lib/validation";
import { Button, TextField, useToast } from "../../ui";
import type { AuthStackParams } from "../navTypes";
import { AuthLayout } from "./AuthLayout";

type Props = NativeStackScreenProps<AuthStackParams, "ResetPassword">;

export default function ResetPasswordScreen({ navigation, route }: Props) {
  const toast = useToast();
  const { email } = route.params;
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<Record<string, string | null>>({});

  const mutation = useMutation({
    mutationFn: () => resetPassword(email, code.trim(), password),
    onSuccess: () => {
      toast.success(t.auth.resetDone);
      navigation.navigate("Login");
    },
    onError: (err) => toast.error(errorMessage(err))
  });

  const submit = () => {
    const next = {
      code: validateRequired(code) ? t.auth.codeRequired : null,
      password: validatePassword(password)
    };
    setErrors(next);
    if (!next.code && !next.password) mutation.mutate();
  };

  return (
    <AuthLayout subtitle={t.auth.resetText} testID="reset-screen" title={t.auth.resetTitle}>
      <TextField
        autoCapitalize="none"
        autoComplete="one-time-code"
        error={errors.code}
        keyboardType="number-pad"
        label={t.auth.code}
        onChangeText={setCode}
        testID="reset-code-input"
        value={code}
      />
      <TextField
        autoCapitalize="none"
        autoComplete="new-password"
        error={errors.password}
        hint={t.auth.passwordHint}
        label={t.auth.newPassword}
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        testID="reset-password-input"
        value={password}
      />
      <Button
        label={t.auth.resetCta}
        loading={mutation.isPending}
        onPress={submit}
        testID="reset-submit-button"
      />
    </AuthLayout>
  );
}
