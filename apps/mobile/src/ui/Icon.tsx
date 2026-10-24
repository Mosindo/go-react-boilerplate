import React from "react";
import { Ionicons } from "@expo/vector-icons";
import { useTheme } from "../theme/theme";

type Props = { name: React.ComponentProps<typeof Ionicons>["name"]; size?: number; color?: string };

export function Icon({ name, size = 24, color }: Props) {
  const { colors } = useTheme();
  return <Ionicons name={name} size={size} color={color ?? colors.text} />;
}
