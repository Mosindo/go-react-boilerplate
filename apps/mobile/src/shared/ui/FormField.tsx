import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Text } from "./Text";

type Props = { label: string; error?: string | null; hint?: string; children: React.ReactNode };

export function FormField({ label, error, hint, children }: Props) {
  return (
    <View style={styles.field}>
      <Text variant="label">{label}</Text>
      {children}
      {error ? (
        <Text variant="caption" tone="danger" accessibilityRole="alert">
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
