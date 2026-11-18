import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Icon, type IconName } from "./Icon";
import { Text } from "./Text";

export function LoadingState({ label }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={styles.center}>
      <ActivityIndicator color={colors.accent} size="large" />
      {label ? (
        <Text color="textMuted" variant="caption">
          {label}
        </Text>
      ) : null}
    </View>
  );
}

type EmptyProps = {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon, title, message, actionLabel, onAction, testID }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <View style={[styles.badge, { backgroundColor: colors.accentSoft }]}>
        <Icon color="accent" name={icon} size={30} />
      </View>
      <Text align="center" variant="title">
        {title}
      </Text>
      {message ? (
        <Text align="center" color="textMuted" style={styles.message}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button fullWidth={false} label={actionLabel} onPress={onAction} variant="secondary" /> : null}
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
      title="Oups"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.sm, padding: spacing.xxl },
  badge: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center", marginBottom: spacing.xs },
  message: { maxWidth: 340 }
});
