import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { confirmPasswordReset, login, register, requestPasswordReset } from "../../api/auth";
import type { AuthStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Input, Screen, Text, useToast } from "../../ui";
import { normalizeEmail, validateEmail, validatePassword } from "../../lib/validation";
import { AuthForm } from "./AuthForm";
import { useAuth } from "./AuthProvider";

type Props<K extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, K>;

function Hero({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <View style={{ marginBottom: spacing.xl, marginTop: spacing.xl }}>
      <Text variant="caption" tone="primary" style={{ letterSpacing: 4, fontWeight: "700" }}>
        LUMEN
      </Text>
      <Text variant="title" style={{ marginTop: spacing.sm }}>
        {title}
      </Text>
      <Text tone="muted" style={{ marginTop: spacing.sm }}>
        {subtitle}
      </Text>
    </View>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { signIn } = useAuth();
  return (
    <Screen scroll>
      <Hero title="Welcome back" subtitle="Sign in to see who is waiting for you. Always free." />
      <AuthForm
        mode="login"
        onSubmit={async (email, password) => signIn(await login(email, password))}
        onForgotPassword={() => navigation.navigate("ForgotPassword")}
        onSwitchMode={() => navigation.navigate("Register")}
      />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { signIn } = useAuth();
  return (
    <Screen scroll>
      <Hero title="Meet someone real" subtitle="No subscriptions, no paywalls, no limits on likes." />
      <AuthForm
        mode="register"
        onSubmit={async (email, password) => signIn(await register(email, password))}
        onSwitchMode={() => navigation.navigate("Login")}
      />
    </Screen>
  );
}

export function ForgotPasswordScreen({ navigation }: Props<"ForgotPassword">) {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const problem = validateEmail(email);
    setError(problem);
    if (problem) return;
    setBusy(true);
    try {
      await requestPasswordReset(normalizeEmail(email));
      navigation.navigate("ResetPassword");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Hero title="Reset your password" subtitle="We will email you a reset code if an account exists." />
      <Input
        label="Email"
        value={email}
        onChangeText={setEmail}
        keyboardType="email-address"
        autoCapitalize="none"
        error={error}
      />
      <Button label="Send reset code" onPress={() => void submit()} loading={busy} />
      <Button
        label="I already have a code"
        variant="ghost"
        onPress={() => navigation.navigate("ResetPassword")}
      />
    </Screen>
  );
}

export function ResetPasswordScreen({ navigation }: Props<"ResetPassword">) {
  const toast = useToast();
  const [token, setToken] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    const problem = !token.trim() ? "Enter the code from your email." : validatePassword(password);
    setError(problem);
    if (problem) return;
    setBusy(true);
    try {
      await confirmPasswordReset(token.trim(), password);
      toast.show("Password updated. Please sign in.");
      navigation.navigate("Login");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Hero title="Choose a new password" subtitle="Paste the code you received, then pick a new password." />
      <Input
        label="Reset code"
        value={token}
        onChangeText={setToken}
        autoCapitalize="none"
        autoCorrect={false}
      />
      <Input
        label="New password"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoCapitalize="none"
        error={error}
      />
      <Button label="Update password" onPress={() => void submit()} loading={busy} />
    </Screen>
  );
}
