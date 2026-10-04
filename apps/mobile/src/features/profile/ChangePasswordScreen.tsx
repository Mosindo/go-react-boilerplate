import React, { useState } from "react";
import { changePassword } from "../../api/auth";
import { getErrorMessage, isApiError } from "../../api/client";
import { validatePassword } from "../../lib/validation";
import { Button, Input, Screen, Text, toast } from "../../ui";

export function ChangePasswordScreen({ navigation }: { navigation: { goBack: () => void } }) {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<{ current?: string; next?: string; confirm?: string; form?: string }>({});
  const [busy, setBusy] = useState(false);

  async function submit() {
    const found = {
      current: current ? undefined : "Enter your current password.",
      next: validatePassword(next) ?? undefined,
      confirm: confirm === next ? undefined : "Passwords don't match."
    };
    if (found.current || found.next || found.confirm) {
      setErrors(found);
      return;
    }
    setErrors({});
    setBusy(true);
    try {
      await changePassword(current, next);
      toast.success("Password changed. Other devices were signed out.");
      navigation.goBack();
    } catch (e) {
      if (isApiError(e) && (e.status === 401 || e.status === 403))
        setErrors({ current: "That isn't your current password." });
      else setErrors({ form: getErrorMessage(e) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Screen keyboardAvoiding scroll>
      <Input
        autoCapitalize="none"
        error={errors.current}
        label="Current password"
        onChangeText={setCurrent}
        passwordToggle
        secureTextEntry
        value={current}
      />
      <Input
        autoCapitalize="none"
        error={errors.next}
        hint="At least 8 characters."
        label="New password"
        onChangeText={setNext}
        passwordToggle
        secureTextEntry
        value={next}
      />
      <Input
        autoCapitalize="none"
        error={errors.confirm}
        label="Confirm new password"
        onChangeText={setConfirm}
        secureTextEntry
        value={confirm}
      />
      {errors.form ? (
        <Text accessibilityRole="alert" tone="danger">
          {errors.form}
        </Text>
      ) : null}
      <Button label="Change password" loading={busy} onPress={submit} />
    </Screen>
  );
}
