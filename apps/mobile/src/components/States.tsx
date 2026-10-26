import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label = "Chargement…" }: { label?: string }) {
  const theme = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator size="large" color={theme.primary} />
      <Text muted>{label}</Text>
    </View>
  );
}

export function EmptyState({
  title,
  message,
  actionLabel,
  onAction,
  emoji = "✦"
}: {
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  emoji?: string;
}) {
  return (
    <View style={styles.center}>
      <Text style={styles.emoji} accessibilityElementsHidden importantForAccessibility="no">
        {emoji}
      </Text>
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text muted center>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} small /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  const theme = useTheme();
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Text
        style={styles.emoji}
        color={theme.danger}
        accessibilityElementsHidden
        importantForAccessibility="no"
      >
        !
      </Text>
      <Text variant="heading" center>
        Une erreur est survenue
      </Text>
      <Text muted center>
        {message ?? "Vérifiez votre connexion et réessayez."}
      </Text>
      {onRetry ? <Button label="Réessayer" onPress={onRetry} small variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  emoji: { fontSize: 40, lineHeight: 48 }
});
