import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text tone="muted" center>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

export function EmptyState({
  icon,
  title,
  message,
  actionLabel,
  onAction
}: {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.halo, { backgroundColor: colors.primarySoft }]}>
        <Icon name={icon} size={30} color={colors.primary} />
      </View>
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" center style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" compact /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return <EmptyState icon="cloud-offline-outline" title="Oups" message={message} actionLabel={onRetry ? "Réessayer" : undefined} onAction={onRetry} />;
}

export function InlineError({ message }: { message?: string | null }) {
  const { colors } = useTheme();
  if (!message) {
    return null;
  }
  return (
    <View style={[styles.inline, { backgroundColor: colors.dangerSoft }]} accessibilityLiveRegion="polite" accessibilityRole="alert">
      <Icon name="alert-circle-outline" size={18} color={colors.danger} />
      <Text variant="caption" tone="danger" style={styles.inlineText}>
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xxl, gap: spacing.sm },
  halo: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  message: { maxWidth: 320, marginBottom: spacing.xs },
  inline: { flexDirection: "row", gap: spacing.xs, alignItems: "center", padding: spacing.sm, borderRadius: 12 },
  inlineText: { flex: 1 }
});
