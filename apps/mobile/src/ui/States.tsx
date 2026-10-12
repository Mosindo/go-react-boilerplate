import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label = "Chargement…" }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator color={colors.primary} size="large" />
      <Text tone="muted">{label}</Text>
    </View>
  );
}

type EmptyProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon = "heart-outline", title, message, actionLabel, onAction }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <Ionicons name={icon} size={52} color={colors.primary} />
      <Text variant="heading" style={styles.text}>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" style={styles.text}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="alert">
      <Ionicons name="cloud-offline-outline" size={48} color={colors.danger} />
      <Text variant="heading" style={styles.text}>
        Oups, ça n'a pas marché
      </Text>
      <Text tone="muted" style={styles.text}>
        {message}
      </Text>
      {onRetry ? <Button label="Réessayer" onPress={onRetry} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  text: { textAlign: "center" }
});
