import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { spacing } from "../../theme/tokens";
import { Button, Header, Screen, Text, TextField, useFeedback } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "DeleteAccount">;

export function DeleteAccountScreen({ navigation }: Props) {
  const { forget } = useSession();
  const { toast, confirm } = useFeedback();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      toast("Votre compte et vos données ont été supprimés.", "success");
      await forget();
    }
  });

  return (
    <Screen scroll>
      <Header title="Supprimer mon compte" onBack={navigation.goBack} />
      <View style={styles.body}>
        <Text>La suppression est immédiate et définitive :</Text>
        <Text tone="muted">
          • votre profil, vos photos, vos likes, vos matchs, vos conversations et vos notifications sont effacés ;{"\n"}• vos sessions sont
          fermées sur tous vos appareils ;{"\n"}• les signalements que vous avez faits restent anonymisés pour la modération.
        </Text>
        <TextField
          label="Mot de passe"
          value={password}
          onChangeText={setPassword}
          secureToggle
          autoComplete="current-password"
          testID="delete-password"
        />
        {remove.error ? <Text tone="danger">{errorMessage(remove.error)}</Text> : null}
        <Button
          label="Supprimer définitivement"
          variant="danger"
          disabled={!password}
          loading={remove.isPending}
          testID="delete-submit"
          onPress={async () => {
            if (
              await confirm({
                title: "Dernière confirmation",
                message: "Cette action est irréversible.",
                confirmLabel: "Supprimer",
                destructive: true
              })
            )
              remove.mutate();
          }}
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: { gap: spacing.md }
});
