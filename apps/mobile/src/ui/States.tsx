import React from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Button } from "./Button";
import { Text } from "./Text";
import { colors, spacing } from "./theme";

type StateProps = {
  icon?: keyof typeof Ionicons.glyphMap;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon = "sparkles-outline", title, message, actionLabel, onAction, testID }: StateProps) {
  return (
    <View style={styles.wrap} testID={testID}>
      <View style={styles.halo}>
        <Ionicons color={colors.accent} name={icon} size={30} />
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

export function ErrorState({ title = "Oups, quelque chose a coincé", message, onAction, actionLabel = "Réessayer" }: Partial<StateProps>) {
  return <EmptyState actionLabel={actionLabel} icon="cloud-offline-outline" message={message} onAction={onAction} title={title} />;
}

export function LoadingState({ label }: { label?: string }) {
  return (
    <View accessibilityLabel={label ?? "Chargement"} accessibilityRole="progressbar" style={styles.wrap}>
      <ActivityIndicator color={colors.accent} size="large" />
      {label ? <Text tone="muted">{label}</Text> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.md,
    padding: spacing.xl
  },
  halo: {
    width: 72,
    height: 72,
    borderRadius: 36,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.accentSoft
  },
  message: {
    maxWidth: 320
  }
});
