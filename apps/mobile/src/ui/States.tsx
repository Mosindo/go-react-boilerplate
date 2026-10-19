import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";
import { errorMessage } from "../lib/errors";
import { Button } from "./Button";
import { Text } from "./Text";

export function Loading({ label = "Chargement…" }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator size="large" color={colors.accent} />
    </View>
  );
}

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction,
  icon
}: {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
  icon?: string;
}) {
  return (
    <View style={styles.center} testID="empty-state">
      {icon ? (
        <Text style={styles.icon} accessibilityElementsHidden importantForAccessibility="no">
          {icon}
        </Text>
      ) : null}
      <Text variant="subtitle" style={styles.centerText}>
        {title}
      </Text>
      <Text tone="muted" style={styles.centerText}>
        {message}
      </Text>
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityRole="alert" testID="error-state">
      <Text variant="subtitle" style={styles.centerText}>
        Oups…
      </Text>
      <Text tone="muted" style={styles.centerText}>
        {errorMessage(error)}
      </Text>
      {onRetry ? <Button label="Réessayer" onPress={onRetry} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  centerText: { textAlign: "center" },
  icon: { fontSize: 44, lineHeight: 52 }
});
