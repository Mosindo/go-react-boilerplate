import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { changePassword } from "../api/auth";
import { errorMessage } from "../api/client";
import { passwordError } from "../domain/validation";
import type { MainScreenProps } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Input, Notice, Text } from "../shared/ui";

export default function ChangePasswordScreen({ navigation }: MainScreenProps<"ChangePassword">) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showErrors, setShowErrors] = useState(false);
  const mutation = useMutation({
    mutationFn: () => changePassword(current, next),
    onSuccess: () => {
      showToast("Password changed. Other devices were signed out.", "success");
      navigation.goBack();
    }
  });

  const problems = {
    current: current.length === 0 ? "Enter your current password." : null,
    next: passwordError(next),
    confirm: confirm !== next ? "The passwords do not match." : null
  };

  const submit = () => {
    setShowErrors(true);
    if (!problems.current && !problems.next && !problems.confirm) {
      mutation.mutate();
    }
  };

  return (
    <ScreenContainer scroll underHeader>
      <Text tone="muted">Changing your password signs you out on your other devices.</Text>
      {mutation.isError ? <Notice kind="error" message={errorMessage(mutation.error)} /> : null}
      <Input
        error={showErrors ? problems.current : null}
        label="Current password"
        onChangeText={setCurrent}
        secureTextEntry
        value={current}
      />
      <Input
        error={showErrors ? problems.next : null}
        hint="At least 8 characters."
        label="New password"
        onChangeText={setNext}
        secureTextEntry
        textContentType="newPassword"
        value={next}
      />
      <Input
        error={showErrors ? problems.confirm : null}
        label="Confirm new password"
        onChangeText={setConfirm}
        onSubmitEditing={submit}
        secureTextEntry
        textContentType="newPassword"
        value={confirm}
      />
      <Button label="Change password" loading={mutation.isPending} onPress={submit} />
    </ScreenContainer>
  );
}
