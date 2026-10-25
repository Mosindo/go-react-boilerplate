import React from "react";
import { StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { spacing, useTheme } from "../../theme";
import { Button } from "./Button";
import type { IconName } from "./IconButton";
import { Text } from "./Text";

type EmptyProps = {
  icon?: IconName;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyState({ icon = "sparkles-outline", title, message, actionLabel, onAction, testID }: EmptyProps) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrapper} testID={testID}>
      <View style={[styles.iconWrap, { backgroundColor: colors.primarySoft }]}>
        <Ionicons color={colors.primary} name={icon} size={32} />
      </View>
      <Text style={styles.centered} variant="heading">
        {title}
      </Text>
      {message ? (
        <Text color="textMuted" style={styles.centered}>
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

type ErrorProps = { message?: string; onRetry: () => void; testID?: string };

export function ErrorState({ message = "Impossible de charger ce contenu.", onRetry, testID }: ErrorProps) {
  return (
    <EmptyState
      actionLabel="Réessayer"
      icon="cloud-offline-outline"
      message={message}
      onAction={onRetry}
      testID={testID}
      title="Oups, un souci"
    />
  );
}

const styles = StyleSheet.create({
  wrapper: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl },
  iconWrap: { width: 72, height: 72, borderRadius: 36, alignItems: "center", justifyContent: "center" },
  centered: { textAlign: "center" }
});
