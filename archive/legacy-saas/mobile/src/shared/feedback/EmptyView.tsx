import React from "react";
import { StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Button, Card, Text, spacing } from "../ui";

type EmptyViewProps = {
  actionLabel?: string;
  message?: string;
  onAction?: () => void;
  style?: StyleProp<ViewStyle>;
  testID?: string;
  title: string;
};

export function EmptyView({ actionLabel, message, onAction, style, testID, title }: EmptyViewProps) {
  return (
    <View style={[styles.wrap, style]} testID={testID}>
      <Card padding="lg" style={styles.card} variant="muted">
        <Text style={styles.title} variant="heading" weight="bold">
          {title}
        </Text>
        {message ? (
          <Text style={styles.message} tone="muted">
            {message}
          </Text>
        ) : null}
        {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} size="sm" variant="secondary" /> : null}
      </Card>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    width: "100%"
  },
  card: {
    width: "100%",
    maxWidth: 520,
    alignSelf: "center",
    alignItems: "center",
    gap: spacing.sm
  },
  title: {
    textAlign: "center"
  },
  message: {
    textAlign: "center"
  }
});
