import React from "react";
import { Pressable, StyleSheet, Text } from "react-native";
import { fontSizes, radii, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  disabled?: boolean;
};

export function Chip({ label, selected = false, onPress, disabled = false }: Props) {
  const theme = useTheme();
  const content = (
    <Text style={[styles.label, { color: selected ? theme.textOnPrimary : theme.text }]}>
      {label}
    </Text>
  );
  const chipStyle = [
    styles.chip,
    {
      backgroundColor: selected ? theme.primary : theme.surface,
      borderColor: selected ? theme.primary : theme.border,
      opacity: disabled ? 0.5 : 1
    }
  ];
  if (!onPress) {
    return (
      <Pressable disabled style={chipStyle}>
        {content}
      </Pressable>
    );
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: selected, disabled }}
      disabled={disabled}
      onPress={onPress}
      style={chipStyle}
    >
      {content}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1,
    borderRadius: radii.pill,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    marginRight: spacing.sm,
    marginBottom: spacing.sm
  },
  label: { fontSize: fontSizes.sm, fontWeight: "600" }
});
