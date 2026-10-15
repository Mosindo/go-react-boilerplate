import React, { useRef, useState } from "react";
import { Pressable, View, type TextInput } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../i18n/errors";
import { APP_NAME } from "../i18n/fr";
import { useAuth } from "../hooks/useAuth";
import { checkPassword, isValidEmail, normalizeEmail } from "../lib/validation";
import type { RootStackParamList } from "../navigation/types";
import { ScreenContainer } from "../shared/layout";
import { Button, FormField, Notice, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Mode = "login" | "register";

export default function AuthScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { signIn, signUp } = useAuth();
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [touched, setTouched] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const isRegister = mode === "register";
  const emailError = touched && !isValidEmail(email) ? "Saisissez une adresse e-mail valide." : null;
  const passwordIssue = checkPassword(password);
  const passwordError =
    touched && isRegister && passwordIssue
      ? passwordIssue === "short"
        ? "Au moins 8 caractères."
        : "72 octets maximum (environ 70 caractères)."
      : touched && !password
        ? "Saisissez votre mot de passe."
        : null;

  const submit = async () => {
    setTouched(true);
    setFormError(null);
    if (!isValidEmail(email) || !password || (isRegister && passwordIssue)) {
      return;
    }
    setSubmitting(true);
    try {
      if (isRegister) {
        await signUp(normalizeEmail(email), password);
      } else {
        await signIn(normalizeEmail(email), password);
      }
    } catch (error) {
      setFormError(errorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <ScreenContainer keyboardAware scroll testID="auth-screen">
      <View style={{ alignItems: "center", gap: spacing.sm, paddingTop: spacing.xxl, paddingBottom: spacing.lg }}>
        <View
          style={{
            width: 72,
            height: 72,
            borderRadius: 24,
            backgroundColor: palette.brand,
            alignItems: "center",
            justifyContent: "center"
          }}
        >
          <Ionicons color={palette.isDark ? "#1B1017" : "#FFFFFF"} name="heart" size={38} />
        </View>
        <Text accessibilityRole="header" tone="brand" variant="title" weight="bold">
          {APP_NAME}
        </Text>
        <Text style={{ textAlign: "center", maxWidth: 300 }} tone="muted">
          Des rencontres simples et sincères. 100 % gratuit : aucun abonnement, aucune option payante.
        </Text>
      </View>

      <View
        accessibilityRole="tablist"
        style={{ flexDirection: "row", backgroundColor: palette.surfaceMuted, borderRadius: radii.pill, padding: 4 }}
      >
        {(["login", "register"] as const).map((value) => {
          const selected = mode === value;
          return (
            <Pressable
              accessibilityRole="tab"
              accessibilityState={{ selected }}
              key={value}
              onPress={() => {
                setMode(value);
                setFormError(null);
                setTouched(false);
              }}
              style={{
                flex: 1,
                minHeight: 44,
                alignItems: "center",
                justifyContent: "center",
                borderRadius: radii.pill,
                backgroundColor: selected ? palette.surface : "transparent"
              }}
              testID={`auth-mode-${value}`}
            >
              <Text tone={selected ? "default" : "muted"} weight="semibold">
                {value === "login" ? "Connexion" : "Créer un compte"}
              </Text>
            </Pressable>
          );
        })}
      </View>

      <View style={{ gap: spacing.md }}>
        <FormField
          autoCapitalize="none"
          autoComplete="email"
          autoCorrect={false}
          error={emailError}
          keyboardType="email-address"
          label="Adresse e-mail"
          onChangeText={setEmail}
          onSubmitEditing={() => passwordRef.current?.focus()}
          returnKeyType="next"
          testID="auth-email-input"
          textContentType="emailAddress"
          value={email}
        />
        <View>
          <FormField
            autoCapitalize="none"
            autoComplete={isRegister ? "new-password" : "current-password"}
            autoCorrect={false}
            error={passwordError}
            hint={isRegister ? "Au moins 8 caractères." : undefined}
            label="Mot de passe"
            onChangeText={setPassword}
            onSubmitEditing={() => void submit()}
            ref={passwordRef}
            returnKeyType="go"
            secureTextEntry={!showPassword}
            testID="auth-password-input"
            textContentType={isRegister ? "newPassword" : "password"}
            value={password}
          />
          <Pressable
            accessibilityLabel={showPassword ? "Masquer le mot de passe" : "Afficher le mot de passe"}
            accessibilityRole="button"
            hitSlop={8}
            onPress={() => setShowPassword((v) => !v)}
            style={{ position: "absolute", right: spacing.md, top: 26, height: 52, justifyContent: "center" }}
          >
            <Ionicons color={palette.textMuted} name={showPassword ? "eye-off-outline" : "eye-outline"} size={22} />
          </Pressable>
        </View>
      </View>

      {formError ? <Notice message={formError} tone="danger" /> : null}

      <Button
        fullWidth
        label={isRegister ? "Créer mon compte" : "Se connecter"}
        loading={submitting}
        onPress={() => void submit()}
        size="lg"
        testID="auth-submit-button"
      />

      {isRegister ? (
        <Text style={{ textAlign: "center" }} tone="subtle" variant="caption">
          En créant un compte, vous confirmez avoir 18 ans ou plus.
        </Text>
      ) : (
        <Button
          label="Mot de passe oublié ?"
          onPress={() => navigation.navigate("ForgotPassword", { email: normalizeEmail(email) || undefined })}
          size="sm"
          testID="auth-forgot-link"
          variant="ghost"
        />
      )}
    </ScreenContainer>
  );
}
