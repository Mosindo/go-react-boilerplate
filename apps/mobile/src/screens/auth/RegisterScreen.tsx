import React, { useState } from "react";
import { errorMessage } from "../../api/client";
import { isValidEmail, passwordError as validatePassword } from "../../domain/validation";
import { useRegister } from "../../hooks/useAuthActions";
import type { AuthScreenProps } from "../../navigation/types";
import { Button, Input, Notice, Text } from "../../shared/ui";
import { AuthLayout } from "./AuthLayout";

export default function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showErrors, setShowErrors] = useState(false);
  const register = useRegister();

  const emailError = showErrors && !isValidEmail(email) ? "Enter a valid email address." : null;
  const passwordProblem = showErrors ? validatePassword(password) : null;

  const submit = () => {
    setShowErrors(true);
    if (!isValidEmail(email) || validatePassword(password)) {
      return;
    }
    register.mutate({ email, password });
  };

  return (
    <AuthLayout subtitle="It takes a minute. Then we will build your profile together." title="Create your account">
      {register.isError ? (
        <Notice kind="error" message={errorMessage(register.error, "Could not create your account.")} />
      ) : null}
      <Input
        autoCapitalize="none"
        autoComplete="email"
        autoCorrect={false}
        error={emailError}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        textContentType="emailAddress"
        value={email}
      />
      <Input
        autoComplete="new-password"
        error={passwordProblem}
        hint="At least 8 characters."
        label="Password"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        secureTextEntry
        textContentType="newPassword"
        value={password}
      />
      <Text tone="muted" variant="caption">
        By continuing you confirm that you are at least 18 years old.
      </Text>
      <Button label="Create account" loading={register.isPending} onPress={submit} />
      <Button label="I already have an account" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </AuthLayout>
  );
}
