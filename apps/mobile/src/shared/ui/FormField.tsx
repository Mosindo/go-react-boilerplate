import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { spacing } from "./tokens";

type FormFieldProps = {
  label: string;
  error?: string | null;
  hint?: string;
  children: ReactNode;
};

/** Label + control + inline error, announced together to screen readers. */
export function FormField({ label, error, hint, children }: FormFieldProps) {
  return (
    <View style={styles.field}>
      <Text variant="label" weight="semibold">
        {label}
      </Text>
      {children}
      {error ? (
        <Text variant="caption" tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : hint ? (
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({ field: { gap: spacing.xs } });
