import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Button, Card, Text, spacing } from "../ui";

type ErrorViewProps = {
  actionLabel?: string;
  message: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  title?: string;
  compact?: boolean;
};

export function ErrorView({
  actionLabel = "Retry",
  compact = false,
  message,
  onAction,
  style,
  testID,
  title = "Something went wrong"
}: ErrorViewProps) {
  return (
    <Card padding={compact ? "sm" : "md"} style={[styles.card, compact ? styles.cardCompact : null, style]} testID={testID}>
      <View style={styles.copy}>
        <Text tone="danger" variant={compact ? "label" : "heading"} weight="bold">
          {title}
        </Text>
        <Text tone="muted" variant={compact ? "caption" : "body"}>
          {message}
        </Text>
      </View>
      {onAction ? <Button label={actionLabel} onPress={onAction} size="sm" variant="outline" /> : null}
    </Card>
  );
}

const styles = StyleSheet.create({
  card: {
    gap: spacing.md
  },
  cardCompact: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between"
  },
  copy: {
    flex: 1,
    gap: spacing.xs
  }
});
