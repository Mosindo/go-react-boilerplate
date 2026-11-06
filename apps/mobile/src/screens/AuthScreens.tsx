import React, { useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../api/errors";
import { authApi } from "../api/services";
import { useAuth } from "../auth/AuthContext";
import { Button, Input, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { isValidEmail, isValidPassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <Screen>
      <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <ScrollView
          contentContainerStyle={{ padding: spacing.xl, gap: spacing.lg, flexGrow: 1, justifyContent: "center" }}
          keyboardShouldPersistTaps="handled"
        >
          <View style={{ maxWidth: 440, width: "100%", alignSelf: "center", gap: spacing.lg }}>{children}</View>
        </ScrollView>
      </KeyboardAvoidingView>
    </Screen>
  );
}

function Brand() {
  const { colors } = useTheme();
  return (
    <View style={{ gap: spacing.xs, marginBottom: spacing.md }}>
      <Text variant="display" color={colors.primary}>
        {fr.appName}
      </Text>
      <Text muted>{fr.tagline}</Text>
    </View>
  );
}

export function LoginScreen({ navigation }: NativeStackScreenProps<AuthStackParamList, "Login">) {
  const auth = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const emailError = submitted && !isValidEmail(email) ? fr.auth.invalidEmail : null;
  const passwordError =
    submitted && (mode === "register" ? !isValidPassword(password) : password.length === 0)
      ? fr.auth.invalidPassword
      : null;

  const submit = async () => {
    setSubmitted(true);
    setFormError(null);
    if (!isValidEmail(email) || (mode === "register" ? !isValidPassword(password) : password.length === 0)) return;
    setBusy(true);
    try {
      if (mode === "login") await auth.login(email, password);
      else await auth.register(email, password);
    } catch (error) {
      setFormError(errorMessage(error));
      setBusy(false);
    }
  };

  return (
    <Shell>
      <Brand />
      <Text variant="title">{mode === "login" ? fr.auth.loginTitle : fr.auth.registerTitle}</Text>
      <Input
        testID="auth-email"
        label={fr.auth.email}
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        textContentType="emailAddress"
        error={emailError}
      />
      <Input
        testID="auth-password"
        label={fr.auth.password}
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoCapitalize="none"
        autoComplete={mode === "login" ? "current-password" : "new-password"}
        textContentType={mode === "login" ? "password" : "newPassword"}
        hint={mode === "register" ? fr.auth.passwordHint : undefined}
        error={passwordError}
        onSubmitEditing={() => void submit()}
      />
      {formError ? (
        <Text testID="auth-error" color="#B3261E" accessibilityRole="alert">
          {formError}
        </Text>
      ) : null}
      <Button
        testID="auth-submit"
        title={mode === "login" ? fr.auth.login : fr.auth.register}
        loading={busy}
        onPress={() => void submit()}
      />
      {mode === "register" ? (
        <Text variant="caption" muted>
          {fr.auth.terms}
        </Text>
      ) : (
        <Button
          testID="auth-forgot"
          title={fr.auth.forgot}
          variant="ghost"
          onPress={() => navigation.navigate("Forgot", { email })}
        />
      )}
      <Button
        testID="auth-toggle"
        variant="ghost"
        title={mode === "login" ? fr.auth.noAccount : fr.auth.haveAccount}
        onPress={() => {
          setMode(mode === "login" ? "register" : "login");
          setSubmitted(false);
          setFormError(null);
        }}
      />
    </Shell>
  );
}

export function ForgotScreen({ navigation, route }: NativeStackScreenProps<AuthStackParamList, "Forgot">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [step, setStep] = useState<"request" | "reset" | "done">("request");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const requestCode = async () => {
    setError(null);
    if (!isValidEmail(email)) return setError(fr.auth.invalidEmail);
    setBusy(true);
    try {
      await authApi.forgot(email.trim());
      setStep("reset");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const resetPassword = async () => {
    setError(null);
    if (!/^\d{6}$/.test(code.trim())) return setError(fr.auth.invalidCode);
    if (!isValidPassword(password)) return setError(fr.auth.invalidPassword);
    setBusy(true);
    try {
      await authApi.reset(email.trim(), code.trim(), password);
      setStep("done");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Shell>
      <Text variant="title">{fr.auth.forgotTitle}</Text>
      {step === "request" ? (
        <>
          <Text muted>{fr.auth.forgotIntro}</Text>
          <Input
            testID="forgot-email"
            label={fr.auth.email}
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            keyboardType="email-address"
          />
          {error ? <Text color="#B3261E">{error}</Text> : null}
          <Button testID="forgot-send" title={fr.auth.sendCode} loading={busy} onPress={() => void requestCode()} />
        </>
      ) : null}
      {step === "reset" ? (
        <>
          <Text muted>{fr.auth.codeSent}</Text>
          <Input testID="forgot-code" label={fr.auth.code} value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={6} />
          <Input
            testID="forgot-password"
            label={fr.auth.newPassword}
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            autoCapitalize="none"
            hint={fr.auth.passwordHint}
          />
          {error ? <Text color="#B3261E">{error}</Text> : null}
          <Button testID="forgot-reset" title={fr.auth.resetPassword} loading={busy} onPress={() => void resetPassword()} />
        </>
      ) : null}
      {step === "done" ? (
        <>
          <Text testID="forgot-done">{fr.auth.resetDone}</Text>
          <Button title={fr.auth.login} onPress={() => navigation.navigate("Login")} />
        </>
      ) : null}
      {step !== "done" ? <Button variant="ghost" title={fr.common.back} onPress={() => navigation.goBack()} /> : null}
    </Shell>
  );
}
