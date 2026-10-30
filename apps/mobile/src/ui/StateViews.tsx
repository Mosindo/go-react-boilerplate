import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button } from "./Button";
import { Text } from "./Text";
import { spacing, useTheme } from "./theme";

type IconName = React.ComponentProps<typeof Ionicons>["name"];

export function LoadingView({ label = "Chargement…" }: { label?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} accessibilityRole="progressbar" accessibilityLabel={label}>
      <ActivityIndicator color={colors.primary} size="large" />
      <Text tone="muted">{label}</Text>
    </View>
  );
}

type MessageProps = {
  icon: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyView({ icon, title, message, actionLabel, onAction, testID }: MessageProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.center} testID={testID}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Ionicons name={icon} size={34} color={colors.primary} />
      </View>
      <Text variant="heading" style={styles.centerText}>
        {title}
      </Text>
      {message ? (
        <Text tone="muted" style={styles.centerText}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} small variant="secondary" /> : null}
    </View>
  );
}

export function ErrorView({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <EmptyView
      icon="cloud-offline-outline"
      title="Un souci est survenu"
      message={message}
      actionLabel={onRetry ? "Réessayer" : undefined}
      onAction={onRetry}
      testID="error-view"
    />
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  centerText: { textAlign: "center" },
  iconWrap: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" }
});
