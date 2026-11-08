import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { spacing, useTheme } from "../theme/theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingView({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text tone="textMuted" style={styles.gap}>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type StateProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon = "sparkles-outline", title, message, actionLabel, onAction }: StateProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={32} color={colors.primary} />
      </View>
      <Text variant="heading" align="center" style={styles.gap}>
        {title}
      </Text>
      {message ? (
        <Text tone="textMuted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <Button title={actionLabel} onPress={onAction} variant="secondary" style={styles.action} />
      ) : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups, un souci"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  iconWrap: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  gap: { marginTop: spacing.md },
  message: { marginTop: spacing.xs, maxWidth: 320 },
  action: { marginTop: spacing.lg, minWidth: 180 }
});
