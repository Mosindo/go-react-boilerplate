import React from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Button } from "../ui/Button";
import { Text } from "../ui/Text";

export type EmptyViewProps = {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyView({ title, message, actionLabel, onAction }: EmptyViewProps) {
  return (
    <View style={styles.wrap}>
      <Text center variant="heading">
        {title}
      </Text>
      <Text center tone="muted">
        {message}
      </Text>
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flex: 1, alignItems: "center", justifyContent: "center", gap: spacing.md, padding: spacing.xl }
});
