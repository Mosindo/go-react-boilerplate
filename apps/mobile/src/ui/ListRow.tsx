import React, { type ReactNode } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { Text } from "./Text";
import { colors, radii, spacing } from "./theme";

type ListRowProps = {
  title: string;
  subtitle?: string;
  icon?: keyof typeof Ionicons.glyphMap;
  leading?: ReactNode;
  trailing?: ReactNode;
  destructive?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function ListRow({ title, subtitle, icon, leading, trailing, destructive, onPress, testID }: ListRowProps) {
  return (
    <Pressable
      accessibilityRole={onPress ? "button" : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
      testID={testID}
    >
      {leading ?? (icon ? <Ionicons color={destructive ? colors.danger : colors.textMuted} name={icon} size={20} /> : null)}
      <View style={styles.body}>
        <Text numberOfLines={1} tone={destructive ? "danger" : "default"} variant="label">
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={2} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {trailing ?? (onPress ? <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} /> : null)}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md,
    minHeight: 56,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  },
  pressed: {
    backgroundColor: colors.surfaceRaised
  },
  body: {
    flex: 1,
    gap: 2
  }
});
