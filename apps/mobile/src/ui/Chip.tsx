import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors, radii, spacing } from "../theme";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, testID }: ChipProps) {
  const content = (
    <>
      {selected ? <Ionicons name="checkmark" size={14} color={colors.onPrimary} /> : null}
      {highlighted && !selected ? <Ionicons name="sparkles" size={12} color={colors.gold} /> : null}
      <Text variant="small" color={selected ? "onPrimary" : highlighted ? "gold" : "text"}>
        {label}
      </Text>
    </>
  );
  const style = [styles.chip, selected && styles.selected, highlighted && !selected && styles.highlighted];
  if (!onPress) {
    return (
      <Pressable disabled style={style} accessible accessibilityLabel={label}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [...style, pressed && styles.pressed]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.xxs,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.xs,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  selected: {
    backgroundColor: colors.primary,
    borderColor: colors.primary
  },
  highlighted: {
    backgroundColor: colors.goldSoft,
    borderColor: "rgba(233,184,114,0.4)"
  },
  pressed: {
    opacity: 0.8
  }
});
