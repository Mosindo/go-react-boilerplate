import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { authApi } from "../../api/auth";
import { errorMessage } from "../../lib/errors";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../session/SessionProvider";
import { radii, spacing, useTheme } from "../../theme";
import { Button, Header, InlineError, Screen, Text, TextField, toast } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "DeleteAccount">;

export function DeleteAccountScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { signOut } = useSession();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    setError(null);
    setLoading(true);
    try {
      await authApi.deleteAccount(password);
      toast("Votre compte et vos données ont été supprimés.", "success");
      await signOut({ remote: false });
    } catch (e) {
      setError(errorMessage(e));
      setLoading(false);
    }
  };

  return (
    <Screen scroll edges={["top", "bottom"]} testID="delete-account-screen">
      <Header title="Supprimer mon compte" onBack={() => navigation.goBack()} />
      <View style={[styles.warning, { backgroundColor: colors.dangerSoft }]}>
        <Text variant="label" color="danger">
          Cette action est définitive.
        </Text>
        <Text variant="caption" color="text">
          Votre profil, vos photos, vos likes, vos matchs, vos conversations et vos notifications seront effacés immédiatement. Vos
          signalements éventuels sont conservés de façon anonyme pour la modération.
        </Text>
      </View>
      <View style={styles.form}>
        <TextField
          label="Confirmez avec votre mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="current-password"
          testID="delete-password"
        />
        <InlineError message={error} />
        <Button
          label="Supprimer définitivement"
          variant="danger"
          onPress={submit}
          loading={loading}
          disabled={!password}
          testID="delete-submit"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  warning: { padding: spacing.md, borderRadius: radii.md, gap: spacing.xs, marginBottom: spacing.lg },
  form: { gap: spacing.md }
});
