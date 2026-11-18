import React from "react";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme, type Palette } from "../theme";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

export function Icon({ name, size = 22, color = "text" }: { name: IconName; size?: number; color?: keyof Palette }) {
  const { colors } = useTheme();
  return <Ionicons accessibilityElementsHidden importantForAccessibility="no" color={colors[color]} name={name} size={size} />;
}
