import React from "react";
import { Pressable, StyleSheet, type StyleProp, type ViewStyle } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { MIN_TOUCH_TARGET, useTheme, type ColorTokens } from "../../theme";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

type Props = {
  icon: IconName;
  label: string;
  onPress: () => void;
  size?: number;
  color?: keyof ColorTokens;
  disabled?: boolean;
  testID?: string;
  style?: StyleProp<ViewStyle>;
};

export function IconButton({ icon, label, onPress, size = 24, color = "text", disabled, testID, style }: Props) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      accessibilityState={{ disabled: Boolean(disabled) }}
      disabled={disabled}
      hitSlop={4}
      onPress={onPress}
      style={({ pressed }) => [styles.base, { opacity: pressed || disabled ? 0.55 : 1 }, style]}
      testID={testID}
    >
      <Ionicons color={colors[color]} name={icon} size={size} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: { minWidth: MIN_TOUCH_TARGET, minHeight: MIN_TOUCH_TARGET, alignItems: "center", justifyContent: "center" }
});
