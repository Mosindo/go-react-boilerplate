import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import * as authApi from "../../api/auth";
import { Button, Screen, Text, TextField, useToast } from "../../components";
import { errorMessage } from "../../lib/format";
import { validateEmail } from "../../lib/validation";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";

export function ForgotPasswordScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "ForgotPassword">) {
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [error, setError] = useState<string | null>(null);
  const mutation = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim()),
    onSuccess: () => {
      toast.show("If this account exists, a code was sent.", "success");
      navigation.navigate("ResetPassword", { email: email.trim() });
    }
  });

  const submit = (): void => {
    const problem = validateEmail(email);
    setError(problem);
    if (!problem) mutation.mutate();
  };

  return (
    <Screen scroll>
      <View style={{ gap: spacing.xl, flex: 1, justifyContent: "center" }}>
        <View style={{ gap: spacing.xs }}>
          <Text variant="title">Reset your password</Text>
          <Text tone="muted">We will email you a short code to choose a new password.</Text>
        </View>
        <TextField label="Email" value={email} onChangeText={setEmail} error={error} autoCapitalize="none" keyboardType="email-address" testID="forgot-email-input" />
        {mutation.isError ? <Text tone="danger">{errorMessage(mutation.error)}</Text> : null}
        <Button title="Send code" onPress={submit} loading={mutation.isPending} testID="forgot-submit" />
        <Button title="I already have a code" variant="ghost" onPress={() => navigation.navigate("ResetPassword", { email })} />
      </View>
    </Screen>
  );
}
