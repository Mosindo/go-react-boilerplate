import React, { forwardRef } from "react";
import { StyleSheet, View, type TextInput } from "react-native";
import { Input, type InputProps } from "./Input";
import { Text } from "./Text";
import { spacing } from "./tokens";

export type FormFieldProps = InputProps & {
  label: string;
  error?: string | null;
  hint?: string;
  /** Shows "n/max" under the field when set. */
  counter?: number;
};

export const FormField = forwardRef<TextInput, FormFieldProps>(function FormField({ label, error, hint, counter, value, ...props }, ref) {
  return (
    <View style={styles.wrap}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <Input ref={ref} accessibilityLabel={label} invalid={Boolean(error)} value={value} {...props} />
      <View style={styles.footer}>
        {error ? (
          <Text accessibilityLiveRegion="polite" style={styles.flex} tone="danger" variant="caption">
            {error}
          </Text>
        ) : hint ? (
          <Text style={styles.flex} tone="subtle" variant="caption">
            {hint}
          </Text>
        ) : (
          <View style={styles.flex} />
        )}
        {counter !== undefined ? (
          <Text tone="subtle" variant="caption">
            {(value ?? "").length}/{counter}
          </Text>
        ) : null}
      </View>
    </View>
  );
});

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs },
  footer: { flexDirection: "row", gap: spacing.sm, minHeight: 16 },
  flex: { flex: 1 }
});
