import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { colors, spacing } from "./tokens";

type ListRowProps = {
  title: string;
  subtitle?: string;
  leading?: ReactNode;
  trailing?: ReactNode;
  onPress?: () => void;
  tone?: "default" | "danger";
  testID?: string;
};

export function ListRow({ leading, onPress, subtitle, testID, title, tone = "default", trailing }: ListRowProps) {
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
      testID={testID}
    >
      {leading}
      <View style={styles.texts}>
        <Text numberOfLines={1} tone={tone === "danger" ? "danger" : "default"} weight="medium">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="label">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing ?? (onPress ? <Text tone="subtle">›</Text> : null)}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  pressed: {
    opacity: 0.7
  },
  texts: {
    flex: 1,
    gap: 2
  }
});
