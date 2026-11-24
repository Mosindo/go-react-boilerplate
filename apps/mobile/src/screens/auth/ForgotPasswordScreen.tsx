import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { Header } from "../../components/Header";
import { Screen } from "../../components/Screen";
import { TextField } from "../../components/TextField";
import { showToast } from "../../components/Toast";
import type { RootStackParamList } from "../../navigation/types";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { isValidEmail, passwordProblem } from "../../utils/validation";

export default function ForgotPasswordScreen() {
  const navigation = useNavigation();
  const route = useRoute<RouteProp<RootStackParamList, "ForgotPassword">>();
  const { signIn } = useAuth();
  const [step, setStep] = useState<"email" | "code">("email");
  const [email, setEmail] = useState(route.params?.email ?? "");
  const [code, setCode] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const requestCode = async () => {
    if (!isValidEmail(email)) {
      setError("Adresse e-mail invalide.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.forgotPassword(email.trim());
      setStep("code");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    const problem = passwordProblem(password);
    if (!/^\d{8}$/.test(code.trim())) {
      setError("Le code contient 8 chiffres.");
      return;
    }
    if (problem) {
      setError(problem);
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await authApi.resetPassword(email.trim(), code.trim(), password);
      showToast("Mot de passe modifié. Bienvenue !", "success");
      await signIn(email, password);
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <Header title="Récupérer mon compte" onBack={() => navigation.goBack()} />
      <View style={styles.form}>
        {step === "email" ? (
          <>
            <AppText color={colors.textMuted}>Saisis l’e-mail de ton compte : nous t’enverrons un code à 8 chiffres, valable 30 minutes.</AppText>
            <TextField label="E-mail" value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" autoComplete="email" />
            {error ? <AppText color={colors.danger}>{error}</AppText> : null}
            <Button label="Recevoir un code" onPress={requestCode} loading={busy} />
          </>
        ) : (
          <>
            <AppText color={colors.textMuted}>
              Si un compte existe pour {email.trim()}, un code vient d’être envoyé. Pense à vérifier tes spams.
            </AppText>
            <TextField label="Code reçu" value={code} onChangeText={setCode} keyboardType="number-pad" maxLength={8} autoComplete="one-time-code" textContentType="oneTimeCode" />
            <TextField label="Nouveau mot de passe" value={password} onChangeText={setPassword} secureToggle autoComplete="new-password" hint="8 caractères minimum, avec une lettre et un chiffre." />
            {error ? <AppText color={colors.danger}>{error}</AppText> : null}
            <Button label="Changer mon mot de passe" onPress={reset} loading={busy} />
            <Button label="Renvoyer un code" variant="ghost" onPress={requestCode} disabled={busy} />
          </>
        )}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.md, marginTop: spacing.md }
});
