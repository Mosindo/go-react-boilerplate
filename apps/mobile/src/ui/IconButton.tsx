import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors } from "../theme";

type IconName = React.ComponentProps<typeof Ionicons>["name"];

type IconButtonProps = {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  background?: string;
  borderColor?: string;
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
  background = colors.surfaceRaised,
  borderColor = colors.border,
  disabled,
  style,
  testID
}: IconButtonProps) {
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={label}
      aria-disabled={!!disabled}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: background, borderColor },
        pressed ? styles.pressed : null,
        disabled ? styles.disabled : null,
        style
      ]}
    >
      <Ionicons name={icon} size={Math.round(size * 0.46)} color={color} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1
  },
  pressed: {
    transform: [{ scale: 0.94 }]
  },
  disabled: {
    opacity: 0.4
  }
});
