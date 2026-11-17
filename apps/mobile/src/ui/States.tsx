import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, spacing } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type IconName = React.ComponentProps<typeof Ionicons>["name"];

export function LoadingState({ label }: { label?: string }) {
  return (
    <View style={styles.center} accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.accent} size="large" />
      {label ? (
        <Text tone="muted" variant="label">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyStateProps = {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon, title, message, actionLabel, onAction, testID }: EmptyStateProps) {
  return (
    <View style={styles.center} testID={testID}>
      <View style={styles.iconHalo}>
        <Ionicons name={icon} size={30} color={colors.accent} />
      </View>
      <Text variant="heading" center>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" center style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} variant="secondary" onPress={onAction} compact /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title="Oups, ça n'a pas marché"
      message={message ?? "Vérifie ta connexion puis réessaie."}
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
    padding: spacing.xl
  },
  iconHalo: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: colors.accentSoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  message: {
    maxWidth: 320,
    marginBottom: spacing.sm
  }
});
