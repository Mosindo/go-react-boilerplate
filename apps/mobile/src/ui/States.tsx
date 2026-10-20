import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label = "Loading…" }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator color={colors.primary} size="large" />
    </View>
  );
}

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction
}: {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View style={styles.center}>
      <Text variant="heading" style={styles.centerText}>
        {title}
      </Text>
      <Text tone="muted" style={[styles.centerText, styles.gap]}>
        {message}
      </Text>
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const message = error instanceof Error ? error.message : "Something went wrong.";
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Text variant="heading" style={styles.centerText}>
        Something went wrong
      </Text>
      <Text tone="muted" style={[styles.centerText, styles.gap]}>
        {message}
      </Text>
      {onRetry ? <Button label="Try again" onPress={onRetry} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  centerText: { textAlign: "center" },
  gap: { marginVertical: spacing.md }
});
