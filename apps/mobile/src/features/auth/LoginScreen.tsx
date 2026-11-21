import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Button, Header, Screen, Text, TextField } from "../../ui";
import { isValidEmail } from "../../utils/format";
import { useAuth } from "./AuthProvider";

type Props = NativeStackScreenProps<RootStackParamList, "Login">;

export function LoginScreen({ navigation }: Props) {
  const styles = useStyles();
  const { signIn } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!isValidEmail(email) || !password) {
      setError("Renseignez votre e-mail et votre mot de passe.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await signIn(email, password);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Screen scroll keyboard footer={<Button testID="login-submit" label="Se connecter" size="lg" loading={loading} onPress={submit} />}>
        <Header back />
        <Text variant="title" accessibilityRole="header">
          Bon retour parmi nous
        </Text>
        <View style={styles.form}>
          <TextField
            testID="login-email"
            label="E-mail"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            textContentType="emailAddress"
            placeholder="vous@exemple.fr"
          />
          <TextField
            testID="login-password"
            label="Mot de passe"
            value={password}
            onChangeText={setPassword}
            secureToggle
            autoComplete="password"
            textContentType="password"
            onSubmitEditing={submit}
            error={error}
          />
          <Button
            label="Mot de passe oublié ?"
            variant="ghost"
            size="sm"
            style={styles.forgot}
            onPress={() => navigation.navigate("ForgotPassword", { email })}
          />
        </View>
      </Screen>
    </>
  );
}

const useStyles = makeStyles((t) => ({
  form: { gap: t.spacing.lg },
  forgot: { alignSelf: "flex-start", paddingHorizontal: 0 }
}));
