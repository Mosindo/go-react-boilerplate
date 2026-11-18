import React, { useRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View } from "react-native";
import { errorMessage } from "../../api/errors";
import { useAuth } from "../../hooks/useAuth";
import type { AuthScreenProps } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Header, Icon, Screen, Text, TextField } from "../../ui";
import { validateEmail, validatePassword } from "../../utils/validation";

export default function RegisterScreen({ navigation }: AuthScreenProps<"Register">) {
  const { signUp } = useAuth();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adult, setAdult] = useState(false);
  const [errors, setErrors] = useState<{ email?: string; password?: string; form?: string }>({});
  const [loading, setLoading] = useState(false);
  const passwordRef = useRef<TextInput>(null);

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      password: validatePassword(password) ?? undefined,
      form: adult ? undefined : "Confirmez que vous avez 18 ans ou plus."
    };
    setErrors(next);
    if (next.email || next.password || next.form) {
      return;
    }
    setLoading(true);
    try {
      await signUp(email, password);
    } catch (e) {
      setErrors({ form: errorMessage(e) });
      setLoading(false);
    }
  };

  return (
    <Screen keyboard scroll testID="register-screen">
      <Header onBack={() => navigation.goBack()} title="Bienvenue" subtitle="Quelques secondes et c’est parti." />
      <View style={{ gap: spacing.md }}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={errors.email}
          inputMode="email"
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          onSubmitEditing={() => passwordRef.current?.focus()}
          returnKeyType="next"
          testID="register-email"
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          ref={passwordRef}
          autoComplete="new-password"
          error={errors.password}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          label="Mot de passe"
          onChangeText={setPassword}
          secureToggle
          testID="register-password"
          textContentType="newPassword"
          value={password}
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: adult }}
          onPress={() => setAdult((v) => !v)}
          style={styles.check}
          testID="register-adult"
        >
          <View
            style={[
              styles.box,
              { borderColor: adult ? colors.accent : colors.borderStrong, backgroundColor: adult ? colors.accent : "transparent" }
            ]}
          >
            {adult ? <Icon color="textOnAccent" name="checkmark" size={16} /> : null}
          </View>
          <Text style={styles.checkLabel}>J’ai 18 ans ou plus et je m’engage à respecter les autres membres.</Text>
        </Pressable>
        {errors.form ? (
          <Text color="danger" testID="register-error" variant="caption">
            {errors.form}
          </Text>
        ) : null}
      </View>
      <Button label="Créer mon compte" loading={loading} onPress={submit} testID="register-submit" />
    </Screen>
  );
}

const styles = StyleSheet.create({
  check: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  box: { width: 24, height: 24, borderRadius: radii.sm / 2, borderWidth: 2, alignItems: "center", justifyContent: "center" },
  checkLabel: { flex: 1 }
});
