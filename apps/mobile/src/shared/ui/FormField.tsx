import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Text } from "./Text";

/** Wraps a custom (non text-input) control with a label and error line. */
export function FormField({ label, error, children }: { label: string; error?: string | null; children: ReactNode }) {
  return (
    <View style={styles.wrap}>
      <Text tone="muted" variant="label">
        {label}
      </Text>
      {children}
      {error ? (
        <Text accessibilityLiveRegion="polite" tone="danger" variant="caption">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.xs } });
