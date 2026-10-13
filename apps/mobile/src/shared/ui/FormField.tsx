import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { spacing } from "./tokens";

type FormFieldProps = {
  label: string;
  hint?: string;
  error?: string | null;
  children: ReactNode;
};

/** Label + hint/error wrapper for custom controls (chips, steppers, toggles). */
export function FormField({ label, hint, error, children }: FormFieldProps) {
  return (
    <View style={styles.root}>
      <Text tone="muted" variant="label" weight="semibold">
        {label}
      </Text>
      {children}
      {error ? (
        <Text tone="danger" variant="caption">
          {error}
        </Text>
      ) : hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({ root: { gap: spacing.xs } });
