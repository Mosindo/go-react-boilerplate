import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { colors, spacing } from "../theme/tokens";
import { AppText } from "./AppText";
import { Button } from "./Button";

export function LoadingState({ label }: { label?: string }) {
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? (
        <AppText variant="caption" color={colors.textMuted}>
          {label}
        </AppText>
      ) : null}
    </View>
  );
}

type EmptyProps = {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon, title, message, actionLabel, onAction, testID }: EmptyProps) {
  return (
    <View style={styles.center} testID={testID}>
      <View style={styles.halo}>
        <Ionicons name={icon} size={34} color={colors.primary} />
      </View>
      <AppText variant="heading" align="center">
        {title}
      </AppText>
      {message ? (
        <AppText color={colors.textMuted} align="center" style={styles.message}>
          {message}
        </AppText>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" compact /> : null}
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
  halo: {
    width: 76,
    height: 76,
    borderRadius: 38,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.xs
  },
  message: { maxWidth: 320, marginBottom: spacing.xs }
});
