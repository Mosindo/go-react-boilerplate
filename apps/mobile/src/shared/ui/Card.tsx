import React from "react";
import { View, type StyleProp, type ViewStyle } from "react-native";
import { radius, spacing, useColors } from "./theme";

export function Card({ children, style }: { children: React.ReactNode; style?: StyleProp<ViewStyle> }) {
  const c = useColors();
  return <View style={[{ backgroundColor: c.surface, borderRadius: radius.lg, borderWidth: 1, borderColor: c.border, padding: spacing.lg }, style]}>{children}</View>;
}
