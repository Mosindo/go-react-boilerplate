import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { Text, spacing } from "../ui";

type HeaderProps = { title: string; subtitle?: string; right?: ReactNode };

/** Large in-screen title used by tab screens (stack screens use the navigator header). */
export function Header({ title, subtitle, right }: HeaderProps) {
  return (
    <View style={styles.row}>
      <View style={styles.titles}>
        <Text accessibilityRole="header" variant="title">
          {title}
        </Text>
        {subtitle ? (
          <Text tone="muted" variant="bodySmall">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: spacing.md,
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.md,
    paddingBottom: spacing.sm
  },
  titles: { flex: 1, gap: 2 }
});
