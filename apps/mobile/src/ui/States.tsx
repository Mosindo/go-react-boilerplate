import React, { type ComponentProps } from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const theme = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={styles.center}>
      <ActivityIndicator color={theme.colors.primary} size="large" />
      {label ? (
        <Text tone="muted" variant="small">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyStateProps = {
  icon: ComponentProps<typeof Ionicons>["name"];
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({ icon, title, message, actionLabel, onAction }: EmptyStateProps) {
  const theme = useTheme();
  return (
    <View style={styles.center}>
      <View style={[styles.badge, { backgroundColor: theme.colors.primarySoft }]}>
        <Ionicons color={theme.colors.primary} name={icon} size={30} />
      </View>
      <Text align="center" variant="title">
        {title}
      </Text>
      {message ? (
        <Text align="center" style={styles.message} tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} size="md" style={styles.action} /> : null}
    </View>
  );
}

export function ErrorState({ message, onRetry }: { message?: string; onRetry?: () => void }) {
  return (
    <EmptyState
      actionLabel={onRetry ? "Réessayer" : undefined}
      icon="cloud-offline-outline"
      message={message ?? "Vérifiez votre connexion puis réessayez."}
      onAction={onRetry}
      title="Oups, un souci"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 32 },
  badge: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: 4 },
  message: { maxWidth: 320 },
  action: { marginTop: 8 }
});
