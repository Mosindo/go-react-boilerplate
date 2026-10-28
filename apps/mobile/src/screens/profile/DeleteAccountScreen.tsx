import React, { useState } from "react";
import { Alert, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { deleteAccount } from "../../api/auth";
import { useAuth } from "../../auth/AuthContext";
import { Button, Screen, Text, TextField } from "../../components";
import { errorMessage } from "../../lib/format";
import { spacing } from "../../theme";

export function DeleteAccountScreen() {
  const { dropSession } = useAuth();
  const [password, setPassword] = useState("");
  const mutation = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: () => dropSession()
  });

  const confirm = (): void => {
    Alert.alert("Delete your account?", "Your profile, photos, matches and messages are permanently erased. This cannot be undone.", [
      { text: "Cancel", style: "cancel" },
      { text: "Delete forever", style: "destructive", onPress: () => mutation.mutate() }
    ]);
  };

  return (
    <Screen scroll testID="delete-account-screen">
      <View style={{ gap: spacing.lg }}>
        <Text variant="heading">Delete my account</Text>
        <Text tone="muted">Enter your password to confirm. Everything linked to your account is permanently erased.</Text>
        <TextField label="Password" value={password} onChangeText={setPassword} secureTextEntry testID="delete-password-input" />
        {mutation.isError ? <Text tone="danger">{errorMessage(mutation.error)}</Text> : null}
        <Button title="Delete my account" variant="danger" onPress={confirm} disabled={!password} loading={mutation.isPending} testID="delete-confirm" />
      </View>
    </Screen>
  );
}
