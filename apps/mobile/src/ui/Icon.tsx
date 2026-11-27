import React from "react";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useTheme } from "../theme/ThemeProvider";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

type IconProps = {
  name: IconName;
  size?: number;
  color?: string;
};

export function Icon({ name, size = 22, color }: IconProps) {
  const { colors } = useTheme();
  return <Ionicons name={name} size={size} color={color ?? colors.text} accessibilityElementsHidden importantForAccessibility="no" />;
}
