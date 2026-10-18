import React from "react";
import { Pressable, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { Text, colors, radii, spacing } from "../shared/ui";

type ChipProps = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function Chip({ label, selected = false, onPress, disabled = false, testID, style }: ChipProps) {
  const body = (
    <Text tone={selected ? "inverse" : "default"} variant="label" weight="semibold">
      {label}
    </Text>
  );
  if (!onPress) {
    return <View style={[styles.chip, styles.static, style]}>{body}</View>;
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.chip,
        selected ? styles.selected : null,
        disabled ? styles.disabled : null,
        pressed ? styles.pressed : null,
        style
      ]}
      testID={testID}
    >
      {body}
    </Pressable>
  );
}

export function ChipRow({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  return <View style={[styles.row, style]}>{children}</View>;
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  chip: {
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundElevated,
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.lg,
    minHeight: 40,
    justifyContent: "center"
  },
  static: {
    backgroundColor: colors.primarySoft,
    borderColor: colors.primaryBorder,
    minHeight: 32,
    paddingVertical: spacing.xs
  },
  selected: { backgroundColor: colors.primary, borderColor: colors.primary },
  disabled: { opacity: 0.45 },
  pressed: { opacity: 0.85 }
});
