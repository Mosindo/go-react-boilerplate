import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { MIN_TOUCH, radii, spacing, useTheme } from "../theme";
import { Text } from "./Text";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string };

export function Chip({ label, selected, onPress, testID }: Props) {
  const { colors } = useTheme();
  const bg = selected ? colors.primary : colors.surfaceMuted;
  const fg = selected ? colors.primaryText : colors.text;
  const content = (
    <Text variant="caption" style={{ color: fg, fontWeight: "600" }}>
      {label}
    </Text>
  );
  if (!onPress) {
    return <Pressable disabled style={[styles.chip, { backgroundColor: bg }]}>{content}</Pressable>;
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      hitSlop={{ top: 6, bottom: 6 }}
      style={[styles.chip, { backgroundColor: bg, minHeight: MIN_TOUCH - 6 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { paddingHorizontal: spacing.md, paddingVertical: spacing.sm, borderRadius: radii.pill, justifyContent: "center" }
});
