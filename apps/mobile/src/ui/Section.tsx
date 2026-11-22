import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { spacing } from "./theme";

export function Section({ title, children, hint }: { title: string; children: ReactNode; hint?: string }) {
  return (
    <View style={styles.wrap}>
      <Text tone="subtle" variant="eyebrow">
        {title}
      </Text>
      <View style={styles.body}>{children}</View>
      {hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: {
    gap: spacing.xs,
    marginBottom: spacing.lg
  },
  body: {
    gap: spacing.xs
  }
});
