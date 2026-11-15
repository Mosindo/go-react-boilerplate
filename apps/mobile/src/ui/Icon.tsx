import React from "react";
import { Ionicons } from "@expo/vector-icons";
import { useTheme, type Palette } from "../theme";

export type IconName = React.ComponentProps<typeof Ionicons>["name"];

type Props = { name: IconName; size?: number; color?: keyof Palette; rawColor?: string };

export function Icon({ name, size = 22, color = "text", rawColor }: Props) {
  const { colors } = useTheme();
  return <Ionicons name={name} size={size} color={rawColor ?? colors[color]} />;
}
