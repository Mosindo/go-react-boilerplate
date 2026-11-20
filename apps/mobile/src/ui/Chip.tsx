import React from "react";
import { Pressable, View } from "react-native";
import { useTheme } from "../theme";
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
  const content = (
    <Text style={{ color: active ? colors.primary : colors.textMuted }} variant="label">
      {label}
    </Text>
  );
  const baseStyle = {
    paddingHorizontal: 14,
    paddingVertical: 8,
    borderRadius: radii.pill,
    borderWidth: 1,
    borderColor: active ? colors.primary : colors.border,
    backgroundColor: active ? colors.primarySoft : colors.surface
  };
  if (!onPress) {
    return <View style={baseStyle}>{content}</View>;
  }
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked: !!selected }}
      accessibilityLabel={label}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [baseStyle, { opacity: pressed ? 0.75 : 1 }]}
    >
      {content}
    </Pressable>
  );
}
