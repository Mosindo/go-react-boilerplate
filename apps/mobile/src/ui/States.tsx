import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={[styles.center, { backgroundColor: colors.background }]}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text tone="muted" variant="caption">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyProps = {
  icon: IconName;
  title: string;
  body?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon, title, body, actionLabel, onAction }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={[styles.center, { backgroundColor: colors.background }]}>
      <View style={[styles.halo, { backgroundColor: colors.primarySoft }]}>
        <Icon color={colors.primary} name={icon} size={30} />
      </View>
      <Text center variant="title">
        {title}
      </Text>
      {body ? (
        <Text center style={styles.body} tone="muted">
          {body}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      actionLabel={onRetry ? "Réessayer" : undefined}
      body={message}
      icon="cloud-offline-outline"
      onAction={onRetry}
      title="Oups"
    />
  );
}

export function InlineError({ message }: { message?: string | null }) {
  const { colors } = useTheme();
  if (!message) {
    return null;
  }
  return (
    <View accessibilityLiveRegion="polite" style={[styles.inline, { backgroundColor: colors.dangerSoft }]}>
      <Icon color={colors.danger} name="alert-circle-outline" size={18} />
      <Text style={styles.flex} tone="danger" variant="caption">
        {message}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xl },
  halo: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  body: { maxWidth: 320, marginBottom: spacing.sm },
  inline: { flexDirection: "row", alignItems: "center", gap: spacing.xs, padding: spacing.sm, borderRadius: 12 },
  flex: { flex: 1 }
});
