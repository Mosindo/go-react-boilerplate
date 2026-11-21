import React from "react";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme, type ThemeColors } from "../theme";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

type IconProps = {
  name: IconName;
  size?: number;
  color?: keyof ThemeColors | string;
};

export function Icon({ name, size = 22, color = "text" }: IconProps) {
  const theme = useTheme();
  const resolved = color in theme.colors ? theme.colors[color as keyof ThemeColors] : color;
  return <Ionicons accessibilityElementsHidden importantForAccessibility="no" name={name} size={size} color={resolved} />;
}
