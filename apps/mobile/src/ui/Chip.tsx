import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { radii, spacing } from "../theme/tokens";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
  variant?: "default" | "onPhoto";
};

export function Chip({ label, selected, onPress, disabled, testID, variant = "default" }: Props) {
  const { colors } = useTheme();
  const onPhoto = variant === "onPhoto";
  const bg = onPhoto ? "rgba(255,255,255,0.18)" : selected ? colors.primary : colors.surface;
  const border = onPhoto ? "rgba(255,255,255,0.35)" : selected ? colors.primary : colors.border;
  const fg = onPhoto ? colors.onPhoto : selected ? colors.onPrimary : colors.text;
  const content = (
    <Text variant="label" style={{ color: fg }}>
      {label}
    </Text>
  );
  if (!onPress) {
    return (
      <Pressable disabled style={[styles.chip, { backgroundColor: bg, borderColor: border }]}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected, disabled: !!disabled }}
      accessibilityLabel={label}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [styles.chip, { backgroundColor: bg, borderColor: border, opacity: disabled ? 0.45 : pressed ? 0.8 : 1 }]}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    paddingHorizontal: spacing.sm + 2,
    paddingVertical: 7,
    borderRadius: radii.pill,
    borderWidth: 1
  }
});
