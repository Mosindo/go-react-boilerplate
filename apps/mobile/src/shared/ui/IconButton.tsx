import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "./theme";
import { radii } from "./tokens";

type IconButtonProps = {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  size?: number;
  color?: string;
  background?: string;
  disabled?: boolean;
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

/** Round icon-only button; `label` is mandatory for screen readers. */
export function IconButton({ icon, label, onPress, size = 52, color, background, disabled, style, testID }: IconButtonProps) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled: !!disabled }}
      disabled={disabled}
      hitSlop={8}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        {
          width: size,
          height: size,
          borderRadius: radii.pill,
          backgroundColor: background ?? colors.surface,
          borderColor: colors.border,
          opacity: disabled ? 0.45 : pressed ? 0.8 : 1
        },
        style
      ]}
    >
      <Ionicons color={color ?? colors.text} name={icon} size={size * 0.46} />
    </Pressable>
  );
}

const styles = StyleSheet.create({ base: { alignItems: "center", justifyContent: "center", borderWidth: 1 } });
