import React, { forwardRef } from "react";
import { View, type TextInput } from "react-native";
import { Input, type InputProps } from "./Input";
import { Text } from "./Text";
import { spacing } from "./tokens";

export type FormFieldProps = InputProps & {
  label: string;
  error?: string | null;
  hint?: string;
};

/** Label + input + inline error/hint, announced together to screen readers. */
export const FormField = forwardRef<TextInput, FormFieldProps>(function FormField(
  { error, hint, label, accessibilityLabel, ...props },
  ref
) {
  return (
    <View style={{ gap: spacing.xs }}>
      <Text tone="muted" variant="label" weight="medium">
        {label}
      </Text>
      <Input ref={ref} accessibilityLabel={accessibilityLabel ?? label} invalid={Boolean(error)} {...props} />
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
});
