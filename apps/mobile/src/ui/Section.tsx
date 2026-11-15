import React from "react";
import { StyleSheet, View } from "react-native";
import { radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

export function Section({ title, children, footer }: { title?: string; children: React.ReactNode; footer?: string }) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      {title ? (
        <Text variant="micro" color="textSubtle" style={styles.title}>
          {title.toUpperCase()}
        </Text>
      ) : null}
      <View style={[styles.card, { backgroundColor: colors.surface, borderColor: colors.border }]}>{children}</View>
      {footer ? (
        <Text variant="caption" color="textSubtle" style={styles.footer}>
          {footer}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.xs, marginBottom: spacing.lg },
  title: { marginLeft: spacing.xs },
  card: { borderRadius: radii.lg, borderWidth: 1, paddingHorizontal: spacing.md, paddingVertical: spacing.xs },
  footer: { marginHorizontal: spacing.xs }
});
