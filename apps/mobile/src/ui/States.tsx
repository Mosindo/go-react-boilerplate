import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} style={styles.center}>
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
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon, title, message, actionLabel, onAction }: EmptyProps) {
  const { colors, spacing } = useTheme();
  return (
    <View style={[styles.center, { padding: spacing.xxl, gap: spacing.sm }]}>
      <View style={[styles.halo, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={34} />
      </View>
      <Text center variant="heading">
        {title}
      </Text>
      {message ? (
        <Text center tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} style={{ marginTop: spacing.sm }} variant="secondary" /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyState
      actionLabel={onRetry ? "Réessayer" : undefined}
      icon="cloud-offline-outline"
      message={message}
      onAction={onRetry}
      title="Un souci de connexion"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12 },
  halo: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: 4 }
});
