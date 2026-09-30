import React, { useRef, useState } from "react";
import { Pressable, StyleSheet, View, type TextInput } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button, Screen, Text, TextField } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { EMAIL_PATTERN, passwordProblem } from "../../lib/format";
import { useSession } from "../../lib/session/SessionProvider";

export default function SignUpScreen() {
  const { signUp } = useSession();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const emailError = touched && !EMAIL_PATTERN.test(email.trim()) ? "Adresse email invalide." : null;
  const passwordError = touched ? passwordProblem(password) : null;

  const submit = async () => {
    setTouched(true);
    if (!EMAIL_PATTERN.test(email.trim()) || passwordProblem(password)) {
      return;
    }
    if (!adult) {
      setError("Lueur est réservée aux personnes majeures.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await signUp(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll testID="signup-screen">
      <Text variant="title">Créer votre compte</Text>
      <Text tone="muted">Quelques secondes suffisent. Vous compléterez votre profil juste après.</Text>
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        error={emailError}
        inputMode="email"
        keyboardType="email-address"
        label="Email"
        onChangeText={setEmail}
        onSubmitEditing={() => passwordRef.current?.focus()}
        returnKeyType="next"
        testID="signup-email"
        textContentType="emailAddress"
        value={email}
      />
      <TextField
        autoComplete="new-password"
        error={passwordError}
        hint="8 caractères minimum, avec au moins une lettre et un chiffre."
        label="Mot de passe"
        onChangeText={setPassword}
        onSubmitEditing={submit}
        ref={passwordRef}
        secureTextEntry
        testID="signup-password"
        textContentType="newPassword"
        value={password}
      />
      <Pressable
        accessibilityRole="checkbox"
        accessibilityState={{ checked: adult }}
        onPress={() => setAdult((v) => !v)}
        style={styles.checkRow}
        testID="signup-adult"
      >
        <Ionicons color={adult ? colors.primary : colors.textSubtle} name={adult ? "checkbox" : "square-outline"} size={24} />
        <Text style={styles.checkText}>J’ai 18 ans ou plus et je m’engage à respecter les autres membres.</Text>
      </Pressable>
      {error ? (
        <Text accessibilityLiveRegion="polite" testID="signup-error" tone="danger">
          {error}
        </Text>
      ) : null}
      <View style={styles.actions}>
        <Button fullWidth label="Créer mon compte" loading={loading} onPress={submit} testID="signup-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  checkRow: { flexDirection: "row", gap: 12, alignItems: "center", paddingVertical: 4 },
  checkText: { flex: 1 },
  actions: { marginTop: 8 }
});
