import { Ionicons } from "@expo/vector-icons";
import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Text } from "./Text";

type StateProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon = "sparkles-outline", title, message, actionLabel, onAction, testID }: StateProps) {
  const { colors, spacing } = useTheme();
  return (
    <View style={[styles.center, { gap: spacing.sm, padding: spacing.xl }]} testID={testID}>
      <View style={[styles.badge, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={30} color={colors.primary} />
      </View>
      <Text variant="title" align="center">
        {title}
      </Text>
      {message ? (
        <Text tone="muted" align="center" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? (
        <Button label={actionLabel} onPress={onAction} style={{ marginTop: spacing.sm }} />
      ) : null}
    </View>
  );
}

export function ErrorState({
  title = "Oups",
  message,
  onAction,
  actionLabel = "Réessayer"
}: Partial<StateProps> & { message: string }) {
  return (
    <EmptyState
      icon="cloud-offline-outline"
      title={title}
      message={message}
      actionLabel={actionLabel}
      onAction={onAction}
    />
  );
}

export function LoadingState({ label }: { label?: string }) {
  const { colors, spacing } = useTheme();
  return (
    <View style={[styles.center, { gap: spacing.sm }]} accessibilityLabel={label ?? "Chargement"}>
      <ActivityIndicator color={colors.primary} size="large" />
      {label ? <Text tone="muted">{label}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  badge: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: 8 },
  message: { maxWidth: 320 }
});
