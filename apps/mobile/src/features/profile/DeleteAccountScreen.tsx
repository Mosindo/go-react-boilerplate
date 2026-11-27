import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/palette";
import { Button, Card, Header, Icon, InlineError, Screen, Text, TextField, toast } from "../../ui";
import type { AppScreenProps } from "../../navigation/types";

export function DeleteAccountScreen({ navigation }: AppScreenProps<"DeleteAccount">) {
  const { colors } = useTheme();
  const { signOut } = useAuth();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      await signOut({ remote: false });
      toast("Ton compte et toutes tes données ont été supprimés.", "success");
    }
  });

  return (
    <Screen scroll keyboard>
      <Header title="Supprimer mon compte" onBack={() => navigation.goBack()} />
      <View style={styles.content}>
        <Card style={[styles.warning, { backgroundColor: colors.dangerSoft, borderColor: colors.dangerSoft }]}>
          <Icon name="warning-outline" size={24} color={colors.danger} />
          <Text tone="danger" style={styles.flex}>
            Cette action est définitive. Ton profil, tes photos, tes matchs, tes conversations et tes notifications seront supprimés
            immédiatement.
          </Text>
        </Card>
        <Text tone="muted">
          Envie d’une pause plutôt ? Tu peux masquer ton profil depuis « Confidentialité » sans rien perdre.
        </Text>
        <TextField label="Confirme avec ton mot de passe" value={password} onChangeText={setPassword} secure autoComplete="password" testID="delete-password" />
        <InlineError message={remove.error ? errorMessage(remove.error) : null} />
        <Button
          label="Supprimer définitivement"
          variant="danger"
          size="lg"
          disabled={!password}
          loading={remove.isPending}
          onPress={() => remove.mutate()}
          testID="delete-confirm"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.md, maxWidth: 560, width: "100%", alignSelf: "center" },
  warning: { flexDirection: "row", gap: spacing.sm, alignItems: "flex-start" },
  flex: { flex: 1 }
});
