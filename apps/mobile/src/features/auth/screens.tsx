import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { APP_NAME, APP_TAGLINE, MIN_AGE } from "../../config";
import { authApi, errorMessage } from "../../api";
import { showToast } from "../../shared/feedback";
import { BrandMark, Button, Screen, Text, TextField } from "../../shared/ui";
import { validateEmail, validatePassword, validateResetCode, normalizeEmail } from "../../shared/validation";
import { spacing } from "../../theme";
import type { AuthStackParamList } from "../../navigation/types";
import { AuthForm, type AuthFormValues } from "./AuthForm";
import { useAuth } from "./AuthProvider";

type Props<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;

export function WelcomeScreen({ navigation }: Props<"Welcome">) {
  return (
    <Screen
      edges={["top", "bottom"]}
      footer={
        <>
          <Button
            label="Créer un compte"
            onPress={() => navigation.navigate("Register")}
            testID="welcome-register-button"
          />
          <Button
            label="J'ai déjà un compte"
            onPress={() => navigation.navigate("Login")}
            testID="welcome-login-button"
            variant="secondary"
          />
          <Text color="textMuted" style={styles.legal} variant="caption">
            {`${APP_NAME} est gratuit, sans abonnement ni limite. Réservé aux personnes de ${MIN_AGE} ans et plus.`}
          </Text>
        </>
      }
      testID="welcome-screen"
    >
      <View style={styles.hero}>
        <BrandMark size={64} />
        <Text color="textMuted" variant="lead">
          {APP_TAGLINE}
        </Text>
      </View>
    </Screen>
  );
}

export function LoginScreen({ navigation }: Props<"Login">) {
  const { signIn } = useAuth();
  const mutation = useMutation({
    mutationFn: authApi.login,
    meta: { silent: true },
    onSuccess: signIn
  });

  return (
    <Screen scroll testID="login-screen">
      <Text variant="title">Bon retour</Text>
      <AuthForm
        loading={mutation.isPending}
        mode="login"
        onSubmit={(values: AuthFormValues) => mutation.mutate(values)}
        submitError={mutation.isError ? errorMessage(mutation.error, "Connexion impossible.") : null}
      />
      <Button
        label="Mot de passe oublié ?"
        onPress={() => navigation.navigate("ForgotPassword")}
        testID="login-forgot-button"
        variant="ghost"
      />
    </Screen>
  );
}

export function RegisterScreen({ navigation }: Props<"Register">) {
  const { signIn } = useAuth();
  const mutation = useMutation({
    mutationFn: authApi.register,
    meta: { silent: true },
    onSuccess: signIn
  });

  return (
    <Screen scroll testID="register-screen">
      <Text variant="title">Créer votre compte</Text>
      <Text color="textMuted">{`Gratuit, sans carte bancaire. Vous devez avoir ${MIN_AGE} ans ou plus.`}</Text>
      <AuthForm
        loading={mutation.isPending}
        mode="register"
        onSubmit={(values: AuthFormValues) => mutation.mutate(values)}
        submitError={mutation.isError ? errorMessage(mutation.error, "Inscription impossible.") : null}
      />
      <Button label="J'ai déjà un compte" onPress={() => navigation.navigate("Login")} variant="ghost" />
    </Screen>
  );
}

export function ForgotPasswordScreen({ navigation, route }: Props<"ForgotPassword">) {
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; code?: string; password?: string }>({});

  const forgot = useMutation({
    mutationFn: authApi.forgotPassword,
    onSuccess: () => {
      setCodeSent(true);
      showToast("Si un compte existe, un code vient d'être envoyé par email.");
    }
  });
  const reset = useMutation({
    mutationFn: authApi.resetPassword,
    meta: { silent: true },
    onSuccess: () => {
      showToast("Mot de passe modifié. Connectez-vous.");
      navigation.navigate("Login");
    }
  });

  const sendCode = () => {
    const emailError = validateEmail(email);
    setErrors({ email: emailError ?? undefined });
    if (!emailError) {
      forgot.mutate(normalizeEmail(email));
    }
  };

  const resetPassword = () => {
    const nextErrors = {
      code: validateResetCode(code) ?? undefined,
      password: validatePassword(password) ?? undefined
    };
    setErrors(nextErrors);
    if (!nextErrors.code && !nextErrors.password) {
      reset.mutate({ email: normalizeEmail(email), code: code.trim(), password });
    }
  };

  return (
    <Screen scroll testID="forgot-screen">
      <Text variant="title">Mot de passe oublié</Text>
      <Text color="textMuted">
        {codeSent
          ? "Saisissez le code reçu par email et choisissez un nouveau mot de passe."
          : "Nous vous enverrons un code à usage unique par email."}
      </Text>
      <TextField
        autoCapitalize="none"
        autoCorrect={false}
        editable={!codeSent}
        error={errors.email}
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        testID="forgot-email-input"
        value={email}
      />
      {codeSent ? (
        <>
          <TextField
            autoCapitalize="none"
            error={errors.code}
            keyboardType="number-pad"
            label="Code de vérification"
            onChangeText={setCode}
            testID="forgot-code-input"
            value={code}
          />
          <TextField
            autoCapitalize="none"
            error={errors.password}
            label="Nouveau mot de passe"
            onChangeText={setPassword}
            secureTextEntry
            testID="forgot-password-input"
            value={password}
          />
          {reset.isError ? (
            <Text color="danger" variant="caption">
              {errorMessage(reset.error, "Réinitialisation impossible.")}
            </Text>
          ) : null}
          <Button
            label="Modifier le mot de passe"
            loading={reset.isPending}
            onPress={resetPassword}
            testID="forgot-reset-button"
          />
          <Button label="Renvoyer un code" loading={forgot.isPending} onPress={sendCode} variant="ghost" />
        </>
      ) : (
        <Button label="Envoyer le code" loading={forgot.isPending} onPress={sendCode} testID="forgot-send-button" />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { flex: 1, justifyContent: "center", gap: spacing.lg },
  legal: { textAlign: "center" }
});
