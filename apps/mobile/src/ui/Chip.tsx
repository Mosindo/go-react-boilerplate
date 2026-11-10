import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  size?: "sm" | "md";
  testID?: string;
};

export function Chip({ label, selected, onPress, disabled, size = "md", testID }: ChipProps) {
  const theme = useTheme();
  const c = theme.colors;
  const content = (
    <Text color={selected ? c.onPrimary : c.text} variant={size === "sm" ? "caption" : "small"} weight="500">
      {label}
    </Text>
  );
  const base = [
    styles.chip,
    {
      paddingHorizontal: size === "sm" ? 10 : 14,
      paddingVertical: size === "sm" ? 5 : 8,
      backgroundColor: selected ? c.primary : c.surfaceAlt,
      borderColor: selected ? c.primary : c.border,
      opacity: disabled ? 0.45 : 1
    }
  ];
  if (!onPress) {
    return (
      <Pressable accessible={false} style={base}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected, disabled: !!disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [base, pressed ? { opacity: 0.75 } : null]}
      testID={testID}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: { borderRadius: 999, borderWidth: 1 }
});
