import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { hitSlop, useTheme } from "../../theme";
import { Text } from "./Text";

export type IconButtonProps = {
  glyph: string;
  label: string;
  onPress: () => void;
  size?: number;
  filled?: boolean;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Round button using a text glyph. Always has at least a 44pt hit target and a spoken label. */
export function IconButton({
  glyph,
  label,
  onPress,
  size = hitSlop,
  filled = false,
  disabled = false,
  style,
  testID
}: IconButtonProps) {
  const theme = useTheme();
  const dimension = Math.max(size, hitSlop);
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled }}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        {
          width: dimension,
          height: dimension,
          borderRadius: dimension / 2,
          backgroundColor: filled ? theme.primary : theme.surface,
          borderColor: filled ? theme.primary : theme.border,
          opacity: disabled ? 0.5 : pressed ? 0.8 : 1
        },
        style
      ]}
      testID={testID}
    >
      <Text style={{ color: filled ? theme.onPrimary : theme.text, fontSize: dimension * 0.42 }} variant="heading">
        {glyph}
      </Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { alignItems: "center", justifyContent: "center", borderWidth: 1 }
});
