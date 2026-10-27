import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { usePalette } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function Loading({ label }: { label?: string }) {
  const p = usePalette();
  return (
    <View
      style={styles.center}
      accessibilityRole="progressbar"
      accessibilityLabel={label ?? "Loading"}
    >
      <ActivityIndicator color={p.primary} size="large" />
      {label ? (
        <Text tone="muted" style={styles.gap}>
          {label}
        </Text>
      ) : null}
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
      <Text variant="heading" style={styles.text}>
        {title}
      </Text>
      <Text tone="muted" style={[styles.text, styles.gap]}>
        {message}
      </Text>
      {actionLabel && onAction ? (
        <Button label={actionLabel} onPress={onAction} variant="secondary" style={styles.gap} />
      ) : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Text variant="heading" style={styles.text}>
        Something went wrong
      </Text>
      <Text tone="muted" style={[styles.text, styles.gap]}>
        {message}
      </Text>
      {onRetry ? (
        <Button label="Try again" onPress={onRetry} variant="secondary" style={styles.gap} />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: 32 },
  text: { textAlign: "center" },
  gap: { marginTop: 12 }
});
