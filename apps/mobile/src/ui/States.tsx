import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";

import { spacing, useTheme } from "../theme";
import { ApiError } from "../api/client";
import { Button } from "./Button";
import { Text } from "./Text";

export function Loading({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View
      style={styles.center}
      accessibilityRole="progressbar"
      accessibilityLabel={label ?? "Chargement"}
    >
      <ActivityIndicator size="large" color={colors.primary} />
      {label ? (
        <Text tone="muted" center>
          {label}
        </Text>
      ) : null}
    </View>
  );
}

export interface EmptyStateProps {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({ icon, title, message, actionLabel, onAction }: EmptyStateProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.badge, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={34} color={colors.primary} />
      </View>
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" center>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <Button label={actionLabel} onPress={onAction} variant="secondary" />
      ) : null}
    </View>
  );
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  return "Une erreur est survenue.";
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const offline = error instanceof ApiError && error.isNetwork;
  return (
    <EmptyState
      icon={offline ? "cloud-offline-outline" : "alert-circle-outline"}
      title={offline ? "Pas de connexion" : "Oups"}
      message={errorMessage(error)}
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
  badge: {
    width: 72,
    height: 72,
    borderRadius: 36,
    alignItems: "center",
    justifyContent: "center",
  },
});
