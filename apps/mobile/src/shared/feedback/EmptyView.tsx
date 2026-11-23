import React from "react";
import { StyleSheet, View } from "react-native";
import { Button, Text, colors, spacing } from "../ui";

type EmptyViewProps = {
  glyph?: string;
  title: string;
  message?: string;
  actionLabel?: string;
  onAction?: () => void;
  testID?: string;
};

export function EmptyView({ actionLabel, glyph = "✦", message, onAction, testID, title }: EmptyViewProps) {
  return (
    <View style={styles.wrap} testID={testID}>
      <View style={styles.glyph}>
        <Text style={styles.glyphText} tone="primary">
          {glyph}
        </Text>
      </View>
      <Text align="center" variant="heading">
        {title}
      </Text>
      {message ? (
        <Text align="center" style={styles.message} tone="muted">
          {message}
        </Text>
      ) : null}
      {actionLabel && onAction ? <Button label={actionLabel} onPress={onAction} size="sm" variant="secondary" /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    padding: spacing.xxl,
    gap: spacing.md
  },
  glyph: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.sm
  },
  glyphText: {
    fontSize: 30,
    lineHeight: 36
  },
  message: {
    maxWidth: 340
  }
});
