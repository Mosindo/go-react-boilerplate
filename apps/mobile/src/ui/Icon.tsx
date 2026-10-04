import React from "react";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useColors } from "../theme";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

type IconProps = { name: IconName; size?: number; color?: string };

/** Decorative icon (hidden from screen readers; the parent control carries the label). */
export function Icon({ name, size = 22, color }: IconProps) {
  const colors = useColors();
  return (
    <Ionicons
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      color={color ?? colors.text}
      name={name}
      size={size}
    />
  );
}
