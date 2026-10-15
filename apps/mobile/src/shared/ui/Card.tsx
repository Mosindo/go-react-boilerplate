import React, { type ReactNode } from "react";
import { Pressable, View, type StyleProp, type ViewStyle } from "react-native";
import { elevation, useTheme } from "../../theme";
import { radii, spacing } from "./tokens";

export type CardProps = {
  children: ReactNode;
  onPress?: () => void;
  padded?: boolean;
  raised?: boolean;
  accessibilityLabel?: string;
  style?: StyleProp<ViewStyle>;
};

export function Card({ accessibilityLabel, children, onPress, padded = true, raised = false, style }: CardProps) {
  const palette = useTheme();
  const base: ViewStyle = {
    backgroundColor: palette.surface,
    borderRadius: radii.lg,
    borderWidth: 1,
    borderColor: palette.border,
    padding: padded ? spacing.lg : 0,
    overflow: "hidden"
  };
  const shadow = raised ? elevation(palette) : null;

  if (onPress) {
    return (
      <Pressable
        accessibilityLabel={accessibilityLabel}
        accessibilityRole="button"
        onPress={onPress}
        style={({ pressed }) => [base, shadow, pressed ? { opacity: 0.88 } : null, style]}
      >
        {children}
      </Pressable>
    );
  }
  return <View style={[base, shadow, style]}>{children}</View>;
}
