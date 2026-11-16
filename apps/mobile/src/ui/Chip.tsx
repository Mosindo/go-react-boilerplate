import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme/tokens";
import { Text } from "./Text";

export type ChipProps = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  icon?: keyof typeof Ionicons.glyphMap;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, icon, testID }: ChipProps) {
  const content = (
    <View style={[styles.chip, selected && styles.selected, highlighted && !selected && styles.highlighted]}>
      {icon ? <Ionicons name={icon} size={14} color={selected ? colors.textOnAccent : highlighted ? colors.accent : colors.textMuted} /> : null}
      <Text variant="caption" style={{ color: selected ? colors.textOnAccent : highlighted ? colors.accent : colors.text }}>
        {label}
      </Text>
    </View>
  );
  if (!onPress) {
    return content;
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => ({ opacity: pressed ? 0.75 : 1 })}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xs,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  selected: { backgroundColor: colors.accent, borderColor: colors.accent },
  highlighted: { borderColor: "rgba(240,138,93,0.5)", backgroundColor: colors.accentSoft }
});
