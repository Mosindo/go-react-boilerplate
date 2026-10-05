import type { ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Text } from "./Text";

/** Label + control + error wrapper for custom (non text-input) controls. */
export function FormField({ label, error, hint, children }: { label: string; error?: string | null; hint?: string; children: ReactNode }) {
  return (
    <View style={styles.wrapper}>
      <Text variant="label" tone="muted">
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

const styles = StyleSheet.create({ wrapper: { gap: spacing.xs } });
