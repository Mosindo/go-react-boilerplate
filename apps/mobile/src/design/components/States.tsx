import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../ThemeProvider";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={styles.center}>
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
  const { colors } = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.badge, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={30} />
      </View>
      <Text align="center" variant="heading">
        {title}
      </Text>
      {message ? (
        <Text align="center" style={styles.message} tone="muted">
          {message}
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
      icon="cloud-offline-outline"
      message={message}
      onAction={onRetry}
      title="Oups, un problème est survenu"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: 32, gap: 12 },
  badge: { width: 68, height: 68, borderRadius: 34, alignItems: "center", justifyContent: "center", marginBottom: 4 },
  message: { maxWidth: 320 }
});
