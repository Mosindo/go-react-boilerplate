import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text, spacing } from "../ui";

export function Section({ title, children, hint }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <View style={styles.section}>
      <Text accessibilityRole="header" tone="muted" variant="eyebrow">
        {title}
      </Text>
      {hint ? (
        <Text tone="subtle" variant="caption">
          {hint}
        </Text>
      ) : null}
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  section: { gap: spacing.sm }
});
