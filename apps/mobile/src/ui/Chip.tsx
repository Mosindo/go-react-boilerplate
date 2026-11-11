import React from "react";
import { Pressable } from "react-native";
import { useTheme } from "../theme/ThemeProvider";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  highlighted?: boolean;
  onPress?: () => void;
  testID?: string;
};

export function Chip({ label, selected, highlighted, onPress, testID }: Props) {
  const { colors, radii } = useTheme();
  const active = selected || highlighted;
  return (
    <Pressable
      testID={testID}
      disabled={!onPress}
      onPress={onPress}
      accessibilityRole={onPress ? "checkbox" : "text"}
      accessibilityState={onPress ? { checked: !!selected } : undefined}
      style={({ pressed }) => ({
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: radii.pill,
        borderWidth: 1.5,
        borderColor: active ? colors.primary : colors.border,
        backgroundColor: selected ? colors.primary : highlighted ? colors.primarySoft : colors.surface,
        opacity: pressed ? 0.8 : 1
      })}
    >
      <Text variant="label" style={{ color: selected ? colors.onPrimary : active ? colors.primary : colors.text }}>
        {label}
      </Text>
    </Pressable>
  );
}
