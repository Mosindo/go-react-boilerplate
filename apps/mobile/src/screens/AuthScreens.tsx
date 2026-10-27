import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { authApi } from "../api/endpoints";
import { validateEmail, validatePassword } from "../lib/validation";
import type { AuthScreenProps } from "../navigation/types";
import { useSession } from "../state/session";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Screen } from "../ui/Screen";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

function Header({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <View style={styles.header}>
      <Text variant="heading" tone="primary" accessibilityRole="header">
        ember
      </Text>
      <Text variant="title">{title}</Text>
      <Text tone="muted">{subtitle}</Text>
    </View>
  );
}

export function LoginScreen({ navigation }: AuthScreenProps<"Login">) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});

  const login = useMutation({
    mutationFn: () => authApi.login(email.trim().toLowerCase(), password),
    onSuccess: signIn,
    onError: (e: Error) => setErrors({ form: e.message })
  });

  const submit = () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: password ? undefined : "Enter your password."
    };
    setErrors(next);
    if (!next.email && !next.password) login.mutate();
  };

  return (
    <Screen scroll>
      <Header title="Welcome back" subtitle="Sign in to continue meeting people." />
      <Input
        label="Email"
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        testID="login-email"
      />
      <Input
        label="Password"
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secureTextEntry
        autoComplete="current-password"
        testID="login-password"
      />
      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert">
          {errors.form}
        </Text>
      ) : null}
      <Button label="Sign in" onPress={submit} loading={login.isPending} testID="login-submit" />
      <Button
        label="Forgot password?"
        variant="ghost"
        onPress={() => navigation.navigate("ForgotPassword", { email })}
      />
      <Button
        label="Create an account"
        variant="secondary"
        onPress={() => navigation.navigate("Register")}
      />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const { signIn } = useSession();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [errors, setErrors] = useState<{
    email?: string;
    password?: string;
    adult?: string;
    form?: string;
  }>({});

  const register = useMutation({
    mutationFn: () => authApi.register(email.trim().toLowerCase(), password),
    onSuccess: signIn,
    onError: (e: Error) => setErrors({ form: e.message })
  });

  const submit = () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      adult: adult ? undefined : "You must confirm you are at least 18."
    };
    setErrors(next);
    if (!next.email && !next.password && !next.adult) register.mutate();
  };

  return (
    <Screen scroll>
      <Header title="Create your account" subtitle="Free forever. No subscriptions, no paywalls." />
      <Input
        label="Email"
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        testID="register-email"
      />
      <Input
        label="Password"
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        hint="At least 8 characters."
        secureTextEntry
        autoComplete="new-password"
        testID="register-password"
      />
      <Button
        label={adult ? "✓ I am at least 18 years old" : "I am at least 18 years old"}
        variant={adult ? "primary" : "secondary"}
        onPress={() => setAdult((v) => !v)}
        testID="register-adult"
      />
      {errors.adult ? (
        <Text variant="caption" tone="danger">
          {errors.adult}
        </Text>
      ) : null}
      {errors.form ? (
        <Text tone="danger" accessibilityRole="alert">
          {errors.form}
        </Text>
      ) : null}
      <Button
        label="Create account"
        onPress={submit}
        loading={register.isPending}
        testID="register-submit"
      />
      <Button
        label="I already have an account"
        variant="ghost"
        onPress={() => navigation.navigate("Login")}
      />
    </Screen>
  );
}

export function ForgotPasswordScreen({ navigation, route }: AuthScreenProps<"ForgotPassword">) {
  const toast = useToast();
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request = useMutation({
    mutationFn: () => authApi.forgotPassword(email.trim().toLowerCase()),
    onSuccess: () => {
      setCodeSent(true);
      setError(null);
    },
    onError: (e: Error) => setError(e.message)
  });
  const reset = useMutation({
    mutationFn: () => authApi.resetPassword(email.trim().toLowerCase(), code.trim(), password),
    onSuccess: () => {
      toast.show("Password updated. Sign in with your new password.", "success");
      navigation.navigate("Login");
    },
    onError: (e: Error) => setError(e.message)
  });

  return (
    <Screen scroll>
      <Header
        title="Reset password"
        subtitle={
          codeSent
            ? "Enter the code we emailed you and choose a new password."
            : "We will email you a reset code."
        }
      />
      <Input
        label="Email"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        keyboardType="email-address"
      />
      {codeSent ? (
        <>
          <Input
            label="Reset code"
            value={code}
            onChangeText={setCode}
            autoCapitalize="characters"
            autoCorrect={false}
          />
          <Input
            label="New password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            hint="At least 8 characters."
          />
        </>
      ) : null}
      {error ? (
        <Text tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      {codeSent ? (
        <Button
          label="Update password"
          loading={reset.isPending}
          onPress={() => {
            const problem = validatePassword(password);
            if (problem) setError(problem);
            else reset.mutate();
          }}
        />
      ) : (
        <Button
          label="Send reset code"
          loading={request.isPending}
          onPress={() => {
            const problem = validateEmail(email);
            if (problem) setError(problem);
            else request.mutate();
          }}
        />
      )}
      <Button
        label="Back to sign in"
        variant="ghost"
        onPress={() => navigation.navigate("Login")}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({ header: { gap: 6, marginTop: 24, marginBottom: 8 } });
