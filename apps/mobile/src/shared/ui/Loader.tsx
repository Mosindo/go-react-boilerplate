import React from "react";
import { ActivityIndicator } from "react-native";
import { useTheme } from "../../theme";

export function Loader({ size = "small" }: { size?: "small" | "large" }) {
  const theme = useTheme();
  return <ActivityIndicator accessibilityLabel="Loading" color={theme.primary} size={size} />;
}
