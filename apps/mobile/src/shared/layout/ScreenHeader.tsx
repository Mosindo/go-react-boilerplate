import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text, spacing } from "../ui";

type ScreenHeaderProps = {
  title: string;
  eyebrow?: string;
  subtitle?: string;
  right?: ReactNode;
};

export function ScreenHeader({ eyebrow, right, subtitle, title }: ScreenHeaderProps) {
  return (
    <View style={styles.row}>
      <View style={styles.texts}>
        {eyebrow ? (
          <Text tone="primary" variant="eyebrow">
            {eyebrow}
          </Text>
        ) : null}
        <Text accessibilityRole="header" variant="title">
          {title}
        </Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: spacing.md
  },
  texts: {
    flex: 1,
    gap: spacing.xxs
  }
});
