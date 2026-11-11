import React, { useState } from "react";
import { View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { authApi } from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { Button, Row, Screen, Section, Text, TextField, useFeedback } from "../../ui";

export function AccountScreen() {
  const { user, signOut, forgetSession } = useAuth();
  const { confirm, toast } = useFeedback();
  const [password, setPassword] = useState("");
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: async () => {
      toast("Ton compte et tes données ont été supprimés.", "success");
      await forgetSession();
    }
  });

  return (
    <Screen edges={[]} testID="account-screen">
      <Section title="Compte">
        <Row icon="mail-outline" label="E-mail" value={user?.email} last />
      </Section>
      <Button
        testID="account-logout"
        label="Se déconnecter"
        variant="secondary"
        icon="log-out-outline"
        onPress={() => void signOut()}
      />
      <Section title="Zone sensible">
        <View style={{ padding: 16, gap: 12 }}>
          <Text tone="muted">
            La suppression est immédiate et définitive : profil, photos, matchs, messages et notifications sont effacés.
          </Text>
          <TextField
            testID="account-delete-password"
            label="Confirme avec ton mot de passe"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
          />
          {remove.isError ? <Text tone="danger">{errorMessage(remove.error)}</Text> : null}
          <Button
            testID="account-delete"
            label="Supprimer mon compte"
            variant="danger"
            icon="trash-outline"
            disabled={!password}
            loading={remove.isPending}
            onPress={async () => {
              if (
                await confirm({
                  title: "Supprimer définitivement ton compte ?",
                  message: "Cette action est irréversible.",
                  confirmLabel: "Supprimer",
                  destructive: true
                })
              ) {
                remove.mutate();
              }
            }}
          />
        </View>
      </Section>
    </Screen>
  );
}
