import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useState } from "react";
import { resetPassword } from "../api/auth";
import { errorMessage } from "../lib/errors";
import { passwordProblem } from "../lib/validation";
import type { RootScreenProps, RootStackParamList } from "../navigation/types";
import { Header, ScreenContainer } from "../shared/layout";
import { showSuccess } from "../shared/feedback";
import { Button, Input, Notice, Text } from "../shared/ui";

/** Opened from the link in the recovery email (deep link or web URL carrying ?token=). */
export default function ResetPasswordScreen({ route }: RootScreenProps<"ResetPassword">) {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const [token, setToken] = useState(route.params?.token ?? "");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    const problem = passwordProblem(password);
    if (problem || !token.trim()) {
      setError(problem ?? "Le lien de réinitialisation est incomplet.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await resetPassword(token.trim(), password);
      showSuccess("Mot de passe modifié. Vous pouvez vous connecter.");
      navigation.navigate("Auth");
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <ScreenContainer scroll testID="reset-screen">
      <Header title="Nouveau mot de passe" onBack={() => navigation.goBack()} />
      {route.params?.token ? null : (
        <Input
          testID="reset-token"
          label="Code reçu par email"
          value={token}
          onChangeText={setToken}
          autoCapitalize="none"
          autoCorrect={false}
        />
      )}
      <Text tone="muted">Choisissez un mot de passe d'au moins 8 caractères. Vous serez déconnecté(e) de vos autres appareils.</Text>
      <Input
        testID="reset-password"
        label="Nouveau mot de passe"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        autoCapitalize="none"
        autoComplete="new-password"
        onSubmitEditing={submit}
      />
      {error ? <Notice kind="error" message={error} /> : null}
      <Button testID="reset-submit" label="Enregistrer" onPress={submit} loading={loading} />
    </ScreenContainer>
  );
}
