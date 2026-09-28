import React, { useState } from "react";
import { Alert } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { deleteAccount } from "../api/auth";
import { errorMessage } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import type { MainScreenProps } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text } from "../shared/ui";

export default function DeleteAccountScreen(_props: MainScreenProps<"DeleteAccount">) {
  const { dropSession } = useAuth();
  const [understood, setUnderstood] = useState(false);
  const [password, setPassword] = useState("");
  const mutation = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      await dropSession();
      showToast("Your account and all your data were deleted.", "info");
    }
  });

  const confirm = () => {
    Alert.alert(
      "Delete your account for good?",
      "This is your last chance. Everything will be erased and cannot be recovered.",
      [
        { text: "Keep my account", style: "cancel" },
        { text: "Delete forever", style: "destructive", onPress: () => mutation.mutate() }
      ]
    );
  };

  return (
    <ScreenContainer scroll underHeader>
      <Text variant="title">Delete your account</Text>
      <Text tone="muted">
        This permanently erases your profile, photos, matches, messages and notifications. It cannot be undone, and
        other people will no longer see you.
      </Text>
      {!understood ? (
        <Button label="I understand, continue" onPress={() => setUnderstood(true)} variant="secondary" />
      ) : (
        <>
          {mutation.isError ? <Notice kind="error" message={errorMessage(mutation.error)} /> : null}
          <Input
            autoComplete="current-password"
            label="Confirm with your password"
            onChangeText={setPassword}
            secureTextEntry
            value={password}
          />
          <Button
            disabled={password.length === 0}
            label="Delete my account"
            loading={mutation.isPending}
            onPress={confirm}
            variant="danger"
          />
        </>
      )}
    </ScreenContainer>
  );
}
