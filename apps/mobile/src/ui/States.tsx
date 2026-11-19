import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { spacing } from "../theme/tokens";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text tone="muted" align="center">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyProps = {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon, title, message, actionLabel, onAction }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.iconCircle, { backgroundColor: colors.primarySoft }]}>
        <Icon name={icon} size={30} color={colors.primary} />
      </View>
      <Text variant="heading" align="center">
        {title}
      </Text>
      {message ? (
        <Text tone="muted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl },
  iconCircle: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  message: { maxWidth: 320, marginBottom: spacing.xs }
});
