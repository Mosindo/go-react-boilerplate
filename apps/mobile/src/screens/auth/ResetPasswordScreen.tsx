import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { Button, Screen, Text, TextField, useToast } from "../../components";
import { errorMessage } from "../../lib/format";
import { validateEmail, validatePassword } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

export function ResetPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "ResetPassword">) {
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string | null; code?: string | null; password?: string | null }>({});
  const mutation = useMutation({
    mutationFn: () => authApi.resetPassword(email.trim(), code.trim(), password),
    onSuccess: () => {
      toast.show("Password updated. Sign in with your new password.", "success");
      navigation.navigate("Login");
    }
  });

  const submit = (): void => {
    const next = {
      email: validateEmail(email),
      code: code.trim().length < 4 ? "Enter the code from your email." : null,
      password: validatePassword(password)
    };
    setErrors(next);
    if (!next.email && !next.code && !next.password) mutation.mutate();
  };

  return (
    <Screen scroll>
      <View style={{ gap: spacing.lg, flex: 1, justifyContent: "center" }}>
        <Text variant="title">Choose a new password</Text>
        <TextField label="Email" value={email} onChangeText={setEmail} error={errors.email} autoCapitalize="none" keyboardType="email-address" />
        <TextField label="Code" value={code} onChangeText={setCode} error={errors.code} autoCapitalize="characters" testID="reset-code-input" />
        <TextField label="New password" value={password} onChangeText={setPassword} error={errors.password} secureTextEntry hint="At least 8 characters." />
        {mutation.isError ? <Text tone="danger">{errorMessage(mutation.error)}</Text> : null}
        <Button title="Update password" onPress={submit} loading={mutation.isPending} testID="reset-submit" />
        <Button title="Back to sign in" variant="ghost" onPress={() => navigation.navigate("Login")} />
      </View>
    </Screen>
  );
}
