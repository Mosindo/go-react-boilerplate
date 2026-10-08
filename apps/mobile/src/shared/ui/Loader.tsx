import React from "react";
import { ActivityIndicator } from "react-native";
import { useTheme } from "../../theme";

export function Loader({ size = "small" }: { size?: "small" | "large" }) {
  const { colors } = useTheme();
  return <ActivityIndicator accessibilityLabel="Chargement" color={colors.primary} size={size} />;
}
