import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { confirmPasswordReset, requestPasswordReset } from "../api/auth";
import { useAuth } from "../hooks/useAuth";
import { strings } from "../i18n/strings";
import { reportError, showToast } from "../shared/feedback";
import { FormScreen } from "../shared/layout";
import { Button, Input, Notice, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme/useTheme";
import { normalizeEmail, validateEmail, validatePassword, validateResetCode } from "../utils/validation";

type Mode = "login" | "register" | "forgot" | "reset";
type Errors = Partial<Record<"email" | "password" | "code", string | null>>;

export default function AuthScreen() {
  const { login, register } = useAuth();
  const { colors } = useTheme();
  const [mode, setMode] = useState<Mode>("register");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [errors, setErrors] = useState<Errors>({});
  const [busy, setBusy] = useState(false);
  const [codeSent, setCodeSent] = useState(false);

  function switchMode(next: Mode) {
    setMode(next);
    setErrors({});
    setPassword("");
  }

  async function submit() {
    const nextErrors: Errors = {};
    if (mode !== "reset") {
      nextErrors.email = validateEmail(email);
    }
    if (mode === "register" || mode === "reset") {
      nextErrors.password = validatePassword(password);
    } else if (mode === "login" && password.length === 0) {
      nextErrors.password = strings.validation.passwordRequired;
    }
    if (mode === "reset") {
      nextErrors.code = validateResetCode(code);
    }
    setErrors(nextErrors);
    if (Object.values(nextErrors).some(Boolean)) {
      return;
    }

    setBusy(true);
    try {
      if (mode === "login") {
        await login(email, password);
      } else if (mode === "register") {
        await register(email, password);
      } else if (mode === "forgot") {
        await requestPasswordReset(normalizeEmail(email));
        setCodeSent(true);
        switchMode("reset");
      } else {
        await confirmPasswordReset(code.trim(), password);
        showToast(strings.auth.resetDone, "success");
        setCode("");
        switchMode("login");
      }
    } catch (error) {
      reportError(error);
    } finally {
      setBusy(false);
    }
  }

  const title =
    mode === "login"
      ? strings.auth.loginTitle
      : mode === "register"
        ? strings.auth.registerTitle
        : mode === "forgot"
          ? strings.auth.forgotTitle
          : strings.auth.resetTitle;
  const submitLabel =
    mode === "login"
      ? strings.auth.loginAction
      : mode === "register"
        ? strings.auth.registerAction
        : mode === "forgot"
          ? strings.auth.forgotAction
          : strings.auth.resetAction;

  return (
    <FormScreen>
      <View style={styles.brand}>
        <View style={[styles.logo, { backgroundColor: colors.primary }]}>
          <Text style={{ color: colors.primaryForeground }} variant="heading" weight="bold">
            {strings.app.name.charAt(0)}
          </Text>
        </View>
        <Text variant="title" weight="bold">
          {strings.app.name}
        </Text>
        <Text tone="muted">{strings.app.tagline}</Text>
      </View>

      <Text variant="heading" weight="bold">
        {title}
      </Text>

      {mode === "forgot" ? <Text tone="muted">{strings.auth.forgotIntro}</Text> : null}
      {mode === "reset" ? (
        <Notice
          description={codeSent ? strings.auth.codeSentBody : strings.auth.haveCodeBody}
          title={strings.auth.codeSentTitle}
        />
      ) : null}

      {mode !== "reset" ? (
        <Input
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
          error={errors.email}
          testID="auth-email-input"
          keyboardType="email-address"
          label={strings.auth.email}
          onChangeText={setEmail}
          textContentType="emailAddress"
          value={email}
        />
      ) : null}

      {mode === "reset" ? (
        <Input
          autoCapitalize="none"
          autoCorrect={false}
          error={errors.code}
          label={strings.auth.resetCode}
          onChangeText={setCode}
          value={code}
        />
      ) : null}

      {mode !== "forgot" ? (
        <Input
          autoCapitalize="none"
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          error={errors.password}
          testID="auth-password-input"
          helperText={mode === "register" || mode === "reset" ? strings.auth.passwordHelp : undefined}
          label={mode === "reset" ? strings.auth.newPassword : strings.auth.password}
          onChangeText={setPassword}
          onSubmitEditing={() => void submit()}
          secureTextEntry
          textContentType={mode === "login" ? "password" : "newPassword"}
          value={password}
        />
      ) : null}

      <Button
        label={submitLabel}
        loading={busy}
        testID="auth-submit-button"
        onPress={() => void submit()}
        size="lg"
      />

      {mode === "register" ? (
        <Text tone="muted" variant="caption">
          {strings.auth.adultsOnly}
        </Text>
      ) : null}

      <View style={styles.links}>
        {mode === "login" ? (
          <>
            <LinkText label={strings.auth.forgotLink} onPress={() => switchMode("forgot")} />
            <LinkText label={strings.auth.toRegister} onPress={() => switchMode("register")} />
          </>
        ) : null}
        {mode === "register" ? (
          <LinkText label={strings.auth.toLogin} onPress={() => switchMode("login")} />
        ) : null}
        {mode === "forgot" ? (
          <>
            <LinkText label={strings.auth.haveCode} onPress={() => switchMode("reset")} />
            <LinkText label={strings.auth.backToLogin} onPress={() => switchMode("login")} />
          </>
        ) : null}
        {mode === "reset" ? (
          <LinkText label={strings.auth.backToLogin} onPress={() => switchMode("login")} />
        ) : null}
      </View>
    </FormScreen>
  );
}

function LinkText({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityRole="link" hitSlop={8} onPress={onPress}>
      <Text tone="primary" weight="semibold">
        {label}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  brand: { alignItems: "flex-start", gap: spacing.xs, marginTop: spacing.xl, marginBottom: spacing.md },
  logo: { width: 48, height: 48, borderRadius: 24, alignItems: "center", justifyContent: "center" },
  links: { gap: spacing.md, alignItems: "flex-start" }
});
