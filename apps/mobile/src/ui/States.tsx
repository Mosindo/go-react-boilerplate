import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, spacing } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type EmptyStateProps = {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  body?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon, title, body, actionLabel, onAction, testID }: EmptyStateProps) {
  return (
    <View style={styles.center} testID={testID}>
      <View style={styles.iconWrap}>
        <Ionicons name={icon} size={30} color={colors.primary} />
      </View>
      <Text variant="heading" align="center">
        {title}
      </Text>
      {body ? (
        <Text variant="small" color="textMuted" align="center" style={styles.body}>
          {body}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" compact /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups, quelque chose bloque"
      body={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
    />
  );
}

export function LoadingState({ label }: { label?: string }) {
  return (
    <View style={styles.center} accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar">
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <Text variant="small" color="textMuted">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.sm,
    padding: spacing.xl
  },
  iconWrap: {
    width: 64,
    height: 64,
    borderRadius: 32,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.xs
  },
  body: {
    maxWidth: 320,
    marginBottom: spacing.xs
  }
});
