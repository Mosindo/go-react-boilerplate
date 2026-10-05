import { useNavigation } from "@react-navigation/native";
import { useState } from "react";
import { forgotPassword } from "../api/auth";
import { errorMessage } from "../lib/errors";
import { isValidEmail, normalizeEmail } from "../lib/validation";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text } from "../shared/ui";

export default function ForgotPasswordScreen() {
  const navigation = useNavigation();
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!isValidEmail(email)) {
      setError("Entrez une adresse email valide.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await forgotPassword(normalizeEmail(email));
      setSent(true);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScreenContainer scroll testID="forgot-screen">
      <Header title="Mot de passe oublié" onBack={() => navigation.goBack()} />
      <Text tone="muted">
        Saisissez l'email de votre compte. Si un compte existe, vous recevrez un lien pour choisir un nouveau mot de passe (valable 1
        heure).
      </Text>
      <Input
        testID="forgot-email"
        label="Email"
        value={email}
        onChangeText={setEmail}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        onSubmitEditing={submit}
      />
      {sent ? (
        <Notice
          kind="success"
          message="Si un compte existe pour cette adresse, un email vient de partir. Pensez à vérifier vos courriers indésirables."
        />
      ) : null}
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="forgot-submit" label="Envoyer le lien" onPress={submit} loading={loading} disabled={sent} />
    </ScreenContainer>
  );
}
