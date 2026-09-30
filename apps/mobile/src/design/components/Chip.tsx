import React from "react";
import { Pressable } from "react-native";
import { useTheme } from "../ThemeProvider";
import { Text } from "./Text";

type Props = {
  label: string;
  selected?: boolean;
  onPress?: () => void;
  highlighted?: boolean;
  testID?: string;
};

export function Chip({ label, selected, onPress, highlighted, testID }: Props) {
  const { colors, radii } = useTheme();
  const active = selected || highlighted;
  return (
    <Pressable
      accessibilityRole={onPress ? "checkbox" : "text"}
      accessibilityState={onPress ? { checked: !!selected } : undefined}
      disabled={!onPress}
      onPress={onPress}
      style={({ pressed }) => ({
        paddingHorizontal: 14,
        paddingVertical: 8,
        borderRadius: radii.pill,
        borderWidth: 1.5,
        borderColor: active ? colors.primary : colors.border,
        backgroundColor: active ? colors.primarySoft : colors.surface,
        opacity: pressed ? 0.75 : 1
      })}
      testID={testID}
    >
      <Text style={{ color: active ? colors.primary : colors.text }} variant="label">
        {label}
      </Text>
    </Pressable>
  );
}
