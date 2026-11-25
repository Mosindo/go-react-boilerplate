import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors } from "../theme";

type IconButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  background?: string;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function IconButton({
  icon,
  label,
  onPress,
  size = 44,
  color = colors.text,
  background = "transparent",
  disabled,
  style,
  testID
}: IconButtonProps) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      testID={testID}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: background, opacity: disabled ? 0.4 : pressed ? 0.7 : 1 },
        style
      ]}
    >
      <Ionicons name={icon} size={Math.round(size * 0.48)} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    alignItems: "center",
    justifyContent: "center"
  }
});
