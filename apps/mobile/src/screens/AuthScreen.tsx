import { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Button, Screen, Text, TextField } from "../components/ui";
import { useToast } from "../components/feedback";
import { errorMessage } from "../lib/errors";
import { validateEmail, validatePassword } from "../lib/validation";
import type { AuthStackParamList } from "../navigation/types";
import { useAuth } from "../state/auth";
import { spacing, useColors } from "../theme";

type Props = NativeStackScreenProps<AuthStackParamList, "Auth">;

export function AuthScreen({ navigation, route }: Props) {
  const { signIn, signUp } = useAuth();
  const toast = useToast();
  const colors = useColors();
  const [mode, setMode] = useState<"login" | "register">(route.params?.mode ?? "login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [busy, setBusy] = useState(false);

  const isRegister = mode === "register";

  const submit = async () => {
    const next = {
      email: validateEmail(email) ?? undefined,
      // En connexion on ne re-valide pas la politique de mot de passe (comptes existants).
      password: isRegister ? (validatePassword(password) ?? undefined) : password ? undefined : "Mot de passe requis",
    };
    setErrors(next);
    if (next.email || next.password) {
      return;
    }
    setBusy(true);
    try {
      if (isRegister) {
        await signUp(email, password);
      } else {
        await signIn(email, password);
      }
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll>
      <View style={{ flex: 1, justifyContent: "center", paddingVertical: spacing.xxl }}>
        <Text style={{ color: colors.primary, marginBottom: spacing.xs }} variant="title">
          Alma
        </Text>
        <Text muted style={{ marginBottom: spacing.xl }}>
          {isRegister ? "Créez votre compte gratuit. Aucune carte bancaire, jamais." : "Content de vous revoir."}
        </Text>

        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={errors.email}
          keyboardType="email-address"
          label="E-mail"
          onChangeText={setEmail}
          textContentType="emailAddress"
          value={email}
        />
        <TextField
          autoCapitalize="none"
          autoComplete={isRegister ? "new-password" : "current-password"}
          error={errors.password}
          label="Mot de passe"
          onChangeText={setPassword}
          onSubmitEditing={submit}
          secure
          textContentType={isRegister ? "newPassword" : "password"}
          value={password}
        />
        {isRegister ? (
          <Text muted style={{ marginBottom: spacing.lg }} variant="caption">
            8 caractères minimum, avec lettres et chiffres. En continuant, vous confirmez avoir 18 ans ou plus.
          </Text>
        ) : null}

        <Button
          loading={busy}
          onPress={submit}
          testID="auth-submit"
          title={isRegister ? "Créer mon compte" : "Se connecter"}
        />
        {!isRegister ? (
          <Button onPress={() => navigation.navigate("ForgotPassword")} title="Mot de passe oublié ?" variant="ghost" />
        ) : null}
        <Button
          onPress={() => {
            setErrors({});
            setMode(isRegister ? "login" : "register");
          }}
          title={isRegister ? "J'ai déjà un compte" : "Je n'ai pas de compte"}
          variant="ghost"
        />
      </View>
    </Screen>
  );
}
