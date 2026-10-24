import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme/theme";
import { ApiError } from "../api/client";
import { Button } from "./Button";
import { Icon } from "./Icon";
import { Text } from "./Text";

export function LoadingState({ label = "Chargement…" }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator size="large" color={colors.primary} />
      <Text muted>{label}</Text>
    </View>
  );
}

type EmptyProps = {
  icon: React.ComponentProps<typeof Icon>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  actionTestID?: string;
  testID?: string;
};

export function EmptyState({
  icon,
  title,
  message,
  actionLabel,
  onAction,
  actionTestID,
  testID,
}: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <Icon name={icon} size={44} color={colors.primary} />
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text muted center>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <Button testID={actionTestID} title={actionLabel} onPress={onAction} variant="secondary" />
      ) : null}
    </View>
  );
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const message = error instanceof ApiError ? error.message : "Une erreur est survenue.";
  return (
    <EmptyState
      testID="error-state"
      icon="cloud-offline-outline"
      title="Oups"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.md,
    padding: spacing.xl,
  },
});
