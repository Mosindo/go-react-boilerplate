import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Text } from "./Text";
import { MIN_TOUCH, radius, spacing, useColors } from "./theme";

type Props = { label: string; selected?: boolean; onPress?: () => void; testID?: string };

export function Chip({ label, selected = false, onPress, testID }: Props) {
  const c = useColors();
  const bg = selected ? c.primary : c.surface;
  const fg = selected ? c.onPrimary : c.text;
  if (!onPress) {
    return (
      <View style={[styles.chip, { backgroundColor: c.surfaceMuted, borderColor: c.surfaceMuted }]}>
        <Text variant="caption" style={{ color: c.text }}>{label}</Text>
      </View>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      hitSlop={{ top: 6, bottom: 6 }}
      style={[styles.chip, styles.tall, { backgroundColor: bg, borderColor: selected ? c.primary : c.border }]}
    >
      <Text variant="label" style={{ color: fg }}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { borderWidth: 1, borderRadius: radius.pill, paddingHorizontal: spacing.md, paddingVertical: spacing.xs, justifyContent: "center" },
  tall: { minHeight: MIN_TOUCH - 8 },
});
