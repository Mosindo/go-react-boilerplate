import React from "react";
import Ionicons from "@expo/vector-icons/Ionicons";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

export function Icon({ name, size = 22, color }: { name: IconName; size?: number; color: string }) {
  return <Ionicons accessibilityElementsHidden importantForAccessibility="no" color={color} name={name} size={size} />;
}
