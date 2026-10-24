import React from "react";
import { Pressable, StyleSheet } from "react-native";
import { radius, spacing, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, onPress, testID }: Props) {
  const { colors } = useTheme();
  const body = (
    <Text variant="label" color={selected ? colors.onPrimary : colors.text}>
      {label}
    </Text>
  );
  const style = [
    styles.chip,
    {
      backgroundColor: selected ? colors.primary : colors.surface,
      borderColor: selected ? colors.primary : colors.border,
    },
  ];
  if (!onPress)
    return (
      <Pressable disabled style={style}>
        {body}
      </Pressable>
    );
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityState={{ selected: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      style={style}
    >
      {body}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  chip: {
    borderWidth: 1,
    borderRadius: radius.pill,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
  },
});
