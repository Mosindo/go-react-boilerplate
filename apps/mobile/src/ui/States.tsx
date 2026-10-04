import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { getErrorMessage } from "../api/client";
import { spacing, useColors } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

type EmptyStateProps = {
  icon?: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
};

export function EmptyState({
  icon = "heart-outline",
  title,
  message,
  actionLabel,
  onAction,
  secondaryLabel,
  onSecondary
}: EmptyStateProps) {
  const colors = useColors();
  return (
    <View style={styles.center}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Icon color={colors.primary} name={icon} size={32} />
      </View>
      <Text center variant="title">
        {title}
      </Text>
      {message ? (
        <Text center tone="muted">
          {message}
        </Text>
      ) : null}
      <View style={styles.actions}>
        {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} /> : null}
        {secondaryLabel && onSecondary ? <Button label={secondaryLabel} onPress={onSecondary} variant="ghost" /> : null}
      </View>
    </View>
  );
}

type ErrorStateProps = { error?: unknown; message?: string; onRetry?: () => void; title?: string };

export function ErrorState({ error, message, onRetry, title = "We couldn't load this" }: ErrorStateProps) {
  const colors = useColors();
  return (
    <View accessibilityRole="alert" style={styles.center}>
      <View style={[styles.iconWrap, { backgroundColor: colors.dangerSoft }]}>
        <Icon color={colors.danger} name="cloud-offline-outline" size={32} />
      </View>
      <Text center variant="title">
        {title}
      </Text>
      <Text center tone="muted">
        {message ?? getErrorMessage(error)}
      </Text>
      {onRetry ? (
        <View style={styles.actions}>
          <Button label="Try again" onPress={onRetry} />
        </View>
      ) : null}
    </View>
  );
}

export function LoadingState({ label = "Loading" }: { label?: string }) {
  const colors = useColors();
  return (
    <View accessibilityLabel={label} accessibilityRole="progressbar" style={styles.center}>
      <ActivityIndicator color={colors.primary} size="large" />
    </View>
  );
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.lg,
    gap: spacing.sm
  },
  iconWrap: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  actions: { alignSelf: "stretch", gap: spacing.xs, marginTop: spacing.sm, maxWidth: 320, width: "100%" }
});
