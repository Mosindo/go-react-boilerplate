import React, { type ReactNode } from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { useTheme } from "./theme";
import { radii, shadows, spacing } from "./tokens";

export type CardProps = {
  children: ReactNode;
  variant?: "elevated" | "muted" | "outline";
  padding?: "none" | "md" | "lg";
  style?: StyleProp<ViewStyle>;
  testID?: string;
};

export function Card({ children, variant = "elevated", padding = "md", style, testID }: CardProps) {
  const { colors } = useTheme();
  return (
    <View
      style={[
        {
          borderRadius: radii.lg,
          padding: padding === "none" ? 0 : padding === "lg" ? spacing.xl : spacing.lg,
          backgroundColor: variant === "muted" ? colors.surfaceMuted : colors.surface,
          borderWidth: variant === "elevated" ? 0 : 1,
          borderColor: colors.border
        },
        variant === "elevated" ? shadows.card : null,
        style
      ]}
      testID={testID}
    >
      {children}
    </View>
  );
}
