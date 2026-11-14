import React, { useRef, useState } from "react";
import { Pressable, StyleSheet, TextInput, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AuthStackParamList } from "../../navigation/types";
import { useAuth } from "../../state/AuthProvider";
import { spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Header, Icon, InlineError, Screen, Text, TextField } from "../../ui";
import { isValidEmail, passwordProblem } from "../../utils/validation";

type Props = NativeStackScreenProps<AuthStackParamList, "Register">;

export function RegisterScreen({ navigation }: Props) {
  const { signIn } = useAuth();
  const { colors } = useTheme();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [accepted, setAccepted] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const passwordRef = useRef<TextInput>(null);
  const register = useMutation({ mutationFn: () => authApi.register(email.trim(), password), onSuccess: signIn });

  const emailError = submitted && !isValidEmail(email) ? "Adresse e-mail invalide." : null;
  const pwProblem = passwordProblem(password);
  const submit = () => {
    setSubmitted(true);
    if (isValidEmail(email) && !pwProblem && accepted) {
      register.mutate();
    }
  };

  return (
    <Screen scroll>
      <Header onBack={navigation.goBack} title="" />
      <Text variant="title">Créer votre compte</Text>
      <Text style={styles.lead} tone="muted">
        Quelques secondes suffisent. Tout est gratuit, pour toujours.
      </Text>
      <View style={styles.form}>
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={emailError}
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
          error={submitted ? pwProblem : null}
          hint="8 caractères minimum, avec au moins une lettre et un chiffre."
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secureToggle
          testID="register-password"
          textContentType="newPassword"
          value={password}
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: accepted }}
          onPress={() => setAccepted((v) => !v)}
          style={styles.consent}
          testID="register-consent"
        >
          <Icon color={accepted ? colors.primary : colors.textSubtle} name={accepted ? "checkbox" : "square-outline"} size={24} />
          <Text style={styles.flex} tone="muted" variant="caption">
            J’ai 18 ans ou plus et je m’engage à respecter les autres membres : bienveillance, consentement et honnêteté.
          </Text>
        </Pressable>
        {submitted && !accepted ? <Text tone="danger" variant="caption">Merci de confirmer cet engagement.</Text> : null}
        <InlineError message={register.isError ? errorMessage(register.error) : null} />
        <Button label="Continuer" loading={register.isPending} onPress={submit} size="lg" testID="register-submit" />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  lead: { marginTop: spacing.xs, marginBottom: spacing.xl },
  form: { gap: spacing.md },
  consent: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  flex: { flex: 1 }
});
