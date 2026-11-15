import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? (
        <Text variant="caption" color="textMuted">
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
  testID?: string;
};

export function EmptyState({ icon, title, message, actionLabel, onAction, testID }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <View style={[styles.halo, { backgroundColor: colors.primarySoft }]}>
        <Icon name={icon} size={34} color="primary" />
      </View>
      <Text variant="title" align="center">
        {title}
      </Text>
      {message ? (
        <Text color="textMuted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} style={styles.action} /> : null}
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

export function InlineError({ message }: { message: string | null }) {
  const { colors } = useTheme();
  if (!message) return null;
  return (
    <View style={[styles.inline, { backgroundColor: colors.dangerSoft }]} accessibilityLiveRegion="polite" accessibilityRole="alert">
      <Icon name="alert-circle-outline" size={18} color="danger" />
      <Text variant="caption" color="danger" style={styles.flex}>
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl, gap: spacing.sm },
  halo: { width: 76, height: 76, borderRadius: 38, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  message: { maxWidth: 320 },
  action: { marginTop: spacing.md, minWidth: 200 },
  inline: { flexDirection: "row", gap: spacing.xs, alignItems: "center", padding: spacing.sm, borderRadius: radii.sm },
  flex: { flex: 1 }
});
