import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { useTheme } from "../theme/theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingView({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? <Text muted>{label}</Text> : null}
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
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View style={styles.center}>
      <Text variant="heading" style={styles.centerText}>
        {title}
      </Text>
      {message ? (
        <Text muted style={styles.centerText}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button title={actionLabel} onPress={onAction} style={styles.action} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Text variant="heading" style={styles.centerText}>
        Oups
      </Text>
      <Text muted style={styles.centerText}>
        {message}
      </Text>
      {onRetry ? <Button title="Réessayer" onPress={onRetry} variant="secondary" style={styles.action} /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: 32, gap: 12 },
  centerText: { textAlign: "center" },
  action: { marginTop: 8, minWidth: 200 }
});
